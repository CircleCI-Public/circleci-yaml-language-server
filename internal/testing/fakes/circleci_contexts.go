package fakes

// This file serves contexts and their environment variables. Two routes report
// the variables: the context list carries them inline when the caller asks with
// include-env-vars, and each context has a route of its own. The list is what
// the language server uses; the real API can refuse the variables while still
// listing the contexts, which is why asking for them is a separate request.

import "net/http"

// fixedTime is the timestamp every context is created at, so that a decoded
// time is predictable in assertions.
const fixedTime = "2026-01-15T10:30:00Z"

// contextsState is the stored contexts and their environment variables.
type contextsState struct {
	contexts map[string][]Context       // org id -> contexts, insertion order
	envVars  map[string][]ContextEnvVar // context id -> env vars, insertion order

	// envVarsRefused refuses a listing that asks for the variables, as the
	// real API does a token that cannot read those of a private context.
	envVarsRefused bool
}

func newContextsState() contextsState {
	return contextsState{
		contexts: map[string][]Context{},
		envVars:  map[string][]ContextEnvVar{},
	}
}

// Context is a stored context.
type Context struct {
	ID    string
	Name  string
	OrgID string
}

// ContextEnvVar is a stored environment variable of a context.
type ContextEnvVar struct {
	Name           string
	TruncatedValue string
}

// AddContext registers a context of an organization. The name is the fully
// qualified one the API reports, for example "my-org/deploy".
func (f *CircleCI) AddContext(orgID, id, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.contexts.contexts[orgID] = append(f.contexts.contexts[orgID], Context{
		ID:    id,
		Name:  name,
		OrgID: orgID,
	})
}

// RefuseContextEnvVars makes a context listing that asks for the environment
// variables answer 403, while one that does not ask still lists the contexts.
func (f *CircleCI) RefuseContextEnvVars() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.contexts.envVarsRefused = true
}

// AddContextEnvVar registers an environment variable of a context.
func (f *CircleCI) AddContextEnvVar(contextID, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.contexts.envVars[contextID] = append(f.contexts.envVars[contextID], ContextEnvVar{
		Name:           name,
		TruncatedValue: "xxxx1234",
	})
}

func (f *CircleCI) handleListContexts(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("owner-id")
	if orgID == "" {
		writeV2Error(w, http.StatusBadRequest, "owner-id is required")

		return
	}

	// Environment variables come back only when they are asked for: a token
	// that can list contexts cannot always read their variables.
	withEnvVars := r.URL.Query().Get("include-env-vars") == "true"

	f.mu.RLock()
	if withEnvVars && f.contexts.envVarsRefused {
		f.mu.RUnlock()
		writeV2Error(w, http.StatusForbidden, "Permission denied")

		return
	}
	items := make([]any, 0, len(f.contexts.contexts[orgID]))
	for _, context := range f.contexts.contexts[orgID] {
		item := map[string]any{
			"id":         context.ID,
			"name":       context.Name,
			"created_at": fixedTime,
		}
		if withEnvVars {
			item["environment_variables"] = f.contextEnvVarEntitiesLocked(context.ID)
		}
		items = append(items, item)
	}
	f.mu.RUnlock()

	f.writeV2Page(w, r, "context", items)
}

func (f *CircleCI) handleListContextEnvVars(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	f.mu.RLock()
	known := false
	for _, contexts := range f.contexts.contexts {
		for _, context := range contexts {
			if context.ID == id {
				known = true
			}
		}
	}
	items := f.contextEnvVarEntitiesLocked(id)
	f.mu.RUnlock()

	if !known {
		writeV2Error(w, http.StatusNotFound, "Context not found")

		return
	}

	f.writeV2Page(w, r, "context/environment-variable", items)
}

// contextEnvVarEntitiesLocked renders a context's environment variables.
// Callers must hold the read lock.
func (f *CircleCI) contextEnvVarEntitiesLocked(contextID string) []any {
	envVars := f.contexts.envVars[contextID]

	entities := make([]any, 0, len(envVars))
	for _, envVar := range envVars {
		entities = append(entities, map[string]any{
			"variable":        envVar.Name,
			"truncated_value": envVar.TruncatedValue,
			"context_id":      contextID,
			"created_at":      fixedTime,
			"updated_at":      fixedTime,
		})
	}

	return entities
}

package fakes

// This file serves self-hosted runner resource classes, which are listed by
// organization id.

import (
	"net/http"
	"strings"
)

// runnerState is the stored runner resource classes.
type runnerState struct {
	classes map[string][]RunnerClass // org id -> resource classes, insertion order
}

func newRunnerState() runnerState {
	return runnerState{classes: map[string][]RunnerClass{}}
}

// RunnerClass is a stored self-hosted runner resource class.
type RunnerClass struct {
	ID            string
	ResourceClass string
	Description   string
}

// AddRunnerResourceClass registers a self-hosted runner resource class of an
// organization, which must also be registered with AddOrg for the route to
// serve it. class is the fully qualified "namespace/class" the API reports;
// its namespace need not match the organization's name.
func (f *CircleCI) AddRunnerResourceClass(orgID, class, description string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.runner.classes[orgID] = append(f.runner.classes[orgID], RunnerClass{
		ID:            "rc-" + strings.ReplaceAll(class, "/", "-"),
		ResourceClass: class,
		Description:   description,
	})
}

func (f *CircleCI) handleListRunnerClasses(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("filter[org_id]")
	if orgID == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Missing Filter", "filter[org_id] is required")

		return
	}

	f.mu.RLock()
	known := f.hasOrgID(orgID)
	classes := f.runner.classes[orgID]
	f.mu.RUnlock()

	// The real route refuses an organization the caller cannot see, rather
	// than reporting it as having no classes.
	if !known {
		writeStatus(w, r.URL.Path, http.StatusForbidden)

		return
	}

	entities := make([]any, 0, len(classes))
	for _, class := range classes {
		entities = append(entities, map[string]any{
			"id": class.ID,
			"attributes": map[string]any{
				"resource_class": class.ResourceClass,
				"description":    class.Description,
			},
			"references": map[string]any{"org": map[string]any{"id": orgID}},
		})
	}

	// The real route answers without a page member.
	writeJSON(w, http.StatusOK, map[string]any{"data": entities})
}

package fakes

// This file serves self-hosted runner resource classes.
//
// Its route says /api/v3, but this is not the CircleCI V3 API: it is the runner
// service's own versioning, which is why it answers with a plain
// {"items": [...]} and not the {"data": ..., "page": ...} envelope the orb and
// catalog routes use. It is served from this fake because a caller reaches it
// with the same token and the same client, and because the language server
// addresses it as runner.<api host>, which a test overrides to point here.
//
// When that call moves onto the standard API — the CLI already calls this path
// on the API host itself — this route moves with it, envelope and all.

import (
	"net/http"
	"strings"
)

// runnerState is the stored runner resource classes.
type runnerState struct {
	classes map[string][]RunnerClass // namespace -> resource classes, insertion order
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

// AddRunnerResourceClass registers a self-hosted runner resource class of a
// namespace. class is the fully qualified "namespace/class" the API reports.
func (f *CircleCI) AddRunnerResourceClass(namespace, class, description string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.runner.classes[namespace] = append(f.runner.classes[namespace], RunnerClass{
		ID:            "rc-" + strings.ReplaceAll(class, "/", "-"),
		ResourceClass: class,
		Description:   description,
	})
}

func (f *CircleCI) handleListRunnerClasses(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")

	f.mu.RLock()
	classes := f.runner.classes[namespace]
	f.mu.RUnlock()

	// An unknown namespace is an empty list, not a 404.
	items := make([]any, 0, len(classes))
	for _, class := range classes {
		items = append(items, map[string]any{
			"id":             class.ID,
			"resource_class": class.ResourceClass,
			"description":    class.Description,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

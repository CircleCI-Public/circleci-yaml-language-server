package fakes

// This file serves self-hosted runner resource classes.
//
// The real route is served by runner.<host> rather than by the API host, so
// reaching it means pointing the caller's runner host at this fake — which
// pkg/server/methods cannot do yet.

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

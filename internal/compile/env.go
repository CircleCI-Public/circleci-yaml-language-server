// Package compile evaluates config the way the reference compiler does:
// templates, logic statements and workflow conditions.
package compile

import (
	"maps"
	"strings"
)

// Env is the variables in scope, keyed by dotted name, such as
// parameters.version or pipeline.git.branch. Values are string, int64,
// bool, nil, []any or map[string]any. An Env is never changed in place.
type Env map[string]any

// With returns env with kv added, replacing any of the same name.
func (e Env) With(kv map[string]any) Env {
	out := maps.Clone(e)
	if out == nil {
		out = Env{}
	}
	maps.Copy(out, kv)
	return out
}

// WithoutPrefix returns env without the names under prefix, as an orb's
// definitions don't see the config's pipeline values.
func (e Env) WithoutPrefix(prefix string) Env {
	return e.filter(func(name string) bool { return !under(name, prefix) })
}

// Only returns the names in env under prefix, as a definition sees only the
// pipeline values of its caller.
func (e Env) Only(prefix string) Env {
	return e.filter(func(name string) bool { return under(name, prefix) })
}

func (e Env) filter(keep func(string) bool) Env {
	out := Env{}
	for name, v := range e {
		if keep(name) {
			out[name] = v
		}
	}
	return out
}

func under(name, prefix string) bool {
	return name == prefix || strings.HasPrefix(name, prefix+".")
}

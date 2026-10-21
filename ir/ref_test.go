package ir

import (
	"encoding/json"
	"maps"
	"regexp"
	"slices"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func fullJob() Job {
	parallelism := 4
	dlc := true
	return Job{
		Version: "job/1",
		Executor: Executor{
			Kind:          "docker",
			ResourceClass: "large",
			Images: []Image{{
				Name:        "cimg/go:1.27",
				Entrypoint:  []string{"/bin/sh"},
				Command:     []string{"-c", "sleep infinity"},
				User:        "circleci",
				Environment: []Env{{Name: "GOFLAGS", Value: "-mod=mod"}},
				Auth:        &ImageAuth{Username: "u", Password: "$DOCKER_PASSWORD"},
			}},
			DLC:    &dlc,
			Origin: []Frame{{Kind: "executor", Name: "go/default", Loc: Loc{Source: "go/orb.yml", Line: 3, Col: 3}}},
		},
		Shell:            "/bin/bash -eo pipefail",
		WorkingDirectory: "~/project",
		Parallelism:      &parallelism,
		Environment:      []Env{{Name: "CGO_ENABLED", Value: "0"}},
		Retention:        "30d",
		Steps: []Step{
			{Kind: "checkout", Origin: []Frame{{Kind: "job", Name: "build", Loc: Loc{Source: "config.yml", Line: 6, Col: 9}}}},
			{
				Kind: "run",
				Name: "test",
				When: "on_fail",
				Args: map[string]any{
					"command":           "go test ./...",
					"no_output_timeout": "10m",
					"background":        false,
					"retries":           int64(-1),
					"ratio":             1.5,
					"paths":             []any{"a", "b"},
					"nested":            map[string]any{"x_y": int64(2)},
				},
				Origin: []Frame{
					{Kind: "job", Name: "build", Loc: Loc{Source: "config.yml", Line: 7, Col: 9}},
					{Kind: "command", Name: "go/test", Loc: Loc{Source: "go/orb.yml", Line: 12, Col: 5}},
				},
			},
		},
		Loc: Loc{Source: "config.yml", Line: 4, Col: 3},
	}
}

func fullGraph(t *testing.T) Graph {
	t.Helper()
	ref, _, err := RefOf("job/1", fullJob())
	assert.NilError(t, err)
	reruns := 0
	return Graph{
		Version: "graph/1",
		Setup:   true,
		Workflows: []Workflow{{
			Name:          "main",
			MaxAutoReruns: &reruns,
			Nodes: []Node{
				{
					Name:     "approve",
					Type:     "approval",
					Upstream: []Edge{{Node: "build-1.27", IfStatus: []string{"success"}}},
					Origin:   []Frame{{Kind: "workflow", Name: "main", Loc: Loc{Source: "config.yml", Line: 20, Col: 9}}},
				},
				{
					Name:           "build-1.27",
					Type:           "build",
					Job:            ref,
					Contexts:       []string{"org-global"},
					SerialGroupIDs: []string{"deploy_group"},
					Matrix:         map[string]string{"go_version": "1.27"},
					Origin:         []Frame{{Kind: "workflow", Name: "main", Loc: Loc{Source: "config.yml", Line: 18, Col: 9}}},
				},
				{
					Name:           "lock-deploy",
					Type:           "lock",
					Key:            "deploy_group",
					SerialGroupIDs: []string{"deploy_group"},
					Origin:         []Frame{},
				},
				{
					Name:     "release",
					Type:     "release",
					Upstream: []Edge{{Node: "approve", IfStatus: []string{"canceled", "failed", "success"}}},
					Release:  &Release{PlanName: "prod", Validation: map[string]any{"max_wait": int64(5)}},
					Origin:   []Frame{},
				},
			},
			Loc: Loc{Source: "config.yml", Line: 16, Col: 3},
		}},
	}
}

func TestGoldenRefs(t *testing.T) {
	// Changing a pinned hash changes the grammar, so it needs a new version.
	t.Run("job", func(t *testing.T) {
		ref, _, err := RefOf("job/1", fullJob())
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(ref, Ref("job/1:sha256:be1b45eae8397df3c76b1377b97fea619da8f0184d369ed58e07fa6184475dab")))
	})
	t.Run("graph", func(t *testing.T) {
		ref, _, err := RefOf("graph/1", fullGraph(t))
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(ref, Ref("graph/1:sha256:4bb56396b2ede2bc85bd8c5b15f7cedacfb0dabc92793cff06bb8472975a571d")))
	})
	t.Run("minimal job", func(t *testing.T) {
		ref, _, err := RefOf("job/1", Job{Version: "job/1", Executor: Executor{Kind: "docker"}})
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(ref, Ref("job/1:sha256:8368be56a1279fc0a8d681743acca3c7e216d108cfe5beff061aff7b2613e649")))
	})
}

func TestRoundTrip(t *testing.T) {
	t.Run("job", func(t *testing.T) {
		want := fullJob()
		b, err := Encode(want)
		assert.NilError(t, err)
		var got Job
		assert.NilError(t, Decode(b, &got))
		assert.Check(t, cmp.DeepEqual(got, want))
	})
	t.Run("graph", func(t *testing.T) {
		want := fullGraph(t)
		b, err := Encode(want)
		assert.NilError(t, err)
		var got Graph
		assert.NilError(t, Decode(b, &got))
		assert.Check(t, cmp.DeepEqual(got, want))
	})
}

func TestMapOrder(t *testing.T) {
	a := map[string]any{}
	b := map[string]any{}
	keys := []string{"alpha", "b", "charlie", "dd", "e", "foxtrot", "g", "hotel"}
	for i, k := range keys {
		a[k] = int64(i)
	}
	for i, k := range slices.Backward(keys) {
		b[k] = int64(i)
	}

	refA, _, err := RefOf("job/1", Step{Kind: "run", Args: a})
	assert.NilError(t, err)
	for range 20 {
		refB, _, err := RefOf("job/1", Step{Kind: "run", Args: maps.Clone(b)})
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(refA, refB))
	}
}

func TestUnset(t *testing.T) {
	unset := Job{Version: "job/1", Executor: Executor{Kind: "docker"}}

	t.Run("absent in JSON", func(t *testing.T) {
		b, err := json.Marshal(unset)
		assert.NilError(t, err)
		var got map[string]any
		assert.NilError(t, json.Unmarshal(b, &got))
		assert.Check(t, cmp.DeepEqual(slices.Sorted(maps.Keys(got)), []string{"executor", "loc", "steps", "version"}))
		assert.Check(t, cmp.DeepEqual(got["executor"], map[string]any{"kind": "docker"}))
	})
	t.Run("absent in CBOR", func(t *testing.T) {
		b, err := Encode(unset)
		assert.NilError(t, err)
		var got map[string]any
		assert.NilError(t, Decode(b, &got))
		assert.Check(t, cmp.DeepEqual(slices.Sorted(maps.Keys(got)), []string{"executor", "loc", "steps", "version"}))
		assert.Check(t, cmp.DeepEqual(got["executor"], map[string]any{"kind": "docker"}))
	})
	t.Run("explicit zero parallelism differs from unset", func(t *testing.T) {
		zero := 0
		set := unset
		set.Parallelism = &zero

		unsetRef, _, err := RefOf("job/1", unset)
		assert.NilError(t, err)
		setRef, b, err := RefOf("job/1", set)
		assert.NilError(t, err)
		assert.Check(t, unsetRef != setRef)

		var got map[string]any
		assert.NilError(t, Decode(b, &got))
		assert.Check(t, cmp.DeepEqual(got["parallelism"], int64(0)))
	})
}

func TestKeys(t *testing.T) {
	snake := regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)
	values := map[string]any{"job": fullJob(), "graph": fullGraph(t)}

	for name, v := range values {
		t.Run(name, func(t *testing.T) {
			j, err := json.Marshal(v)
			assert.NilError(t, err)
			var fromJSON any
			assert.NilError(t, json.Unmarshal(j, &fromJSON))
			jsonKeys := keysOf(fromJSON)

			c, err := Encode(v)
			assert.NilError(t, err)
			var fromCBOR any
			assert.NilError(t, Decode(c, &fromCBOR))

			assert.Check(t, cmp.DeepEqual(keysOf(fromCBOR), jsonKeys), "CBOR keys must match JSON keys")
			for _, k := range jsonKeys {
				assert.Check(t, snake.MatchString(k), "key %q is not snake_case", k)
			}
		})
	}
}

func keysOf(v any) []string {
	seen := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			for k, e := range v {
				seen[k] = true
				walk(e)
			}
		case []any:
			for _, e := range v {
				walk(e)
			}
		}
	}
	walk(v)
	return slices.Sorted(maps.Keys(seen))
}

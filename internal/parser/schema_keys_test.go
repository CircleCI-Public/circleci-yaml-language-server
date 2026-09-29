package parser

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	schema "github.com/CircleCI-Public/circleci-yaml-language-server"
)

func TestClosestKey(t *testing.T) {
	allowed := []string{"name", "command", "shell", "resource_class", "parallelism", "paths", "path", "tag", "tags"}

	testCases := []struct {
		name string
		want string
	}{
		{name: "resouce_class", want: "resource_class"},
		{name: "comand", want: "command"},
		{name: "paralellism", want: "parallelism"},
		{name: "Shell", want: "shell"},
		{name: "nmae", want: "name"},
		{name: "pat", want: "path"},
		// As close to tag as to tags.
		{name: "tagx", want: ""},
		{name: "banana", want: ""},
		// Too short for any edit to be a likely typo.
		{name: "ab", want: ""},
		// Allowed elsewhere, so not a typo of itself.
		{name: "name", want: ""},
	}

	for _, tc := range testCases {
		got := closestKey(tc.name, allowed)
		assert.Check(t, cmp.Equal(got, tc.want), "closest to %q", tc.name)
	}
}

func TestSchemaKeysAt(t *testing.T) {
	keys := newSchemaKeys(schema.EmbeddedSchemaJSON)
	assert.Assert(t, keys != nil)

	testCases := []struct {
		name   string
		fields []string
		has    []string
	}{
		{name: "a job", fields: []string{"jobs", "build"}, has: []string{"resource_class", "steps", "docker"}},
		{name: "an executor", fields: []string{"executors", "e"}, has: []string{"resource_class", "docker"}},
		{name: "a Docker image", fields: []string{"jobs", "build", "docker", "0"}, has: []string{"image", "entrypoint", "auth"}},
		{name: "a run step", fields: []string{"jobs", "build", "steps", "0", "run"}, has: []string{"command", "shell"}},
		{name: "a deploy step", fields: []string{"jobs", "build", "steps", "0", "deploy"}, has: []string{"command", "shell"}},
		{name: "the root", fields: []string{"(root)"}, has: []string{"version", "jobs", "workflows"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := keys.at(tc.fields)
			for _, key := range tc.has {
				assert.Check(t, cmp.Contains(got, key))
			}
		})
	}

	t.Run("none from a schema that isn't JSON", func(t *testing.T) {
		got := newSchemaKeys([]byte("not json")).at([]string{"jobs"})
		assert.Check(t, cmp.Len(got, 0))
	})
}

func TestSchemaKeysFromEachLoader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schema.json")
	assert.NilError(t, os.WriteFile(path, schema.EmbeddedSchemaJSON, 0o600))

	loaders := map[string]func(*JSONSchemaValidator) error{
		"embedded":    (*JSONSchemaValidator).LoadEmbeddedJsonSchema,
		"from bytes":  func(v *JSONSchemaValidator) error { return v.LoadJsonSchemaFromBytes(schema.EmbeddedSchemaJSON) },
		"from a file": func(v *JSONSchemaValidator) error { return v.LoadJsonSchema(path) },
	}

	for name, load := range loaders {
		t.Run(name, func(t *testing.T) {
			validator := JSONSchemaValidator{}
			assert.NilError(t, load(&validator))
			got := validator.keys.at([]string{"jobs", "build"})
			assert.Check(t, cmp.Contains(got, "resource_class"))
		})
	}
}

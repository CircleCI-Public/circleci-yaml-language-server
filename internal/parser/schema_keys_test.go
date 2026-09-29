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

func TestSchemaKeyDescription(t *testing.T) {
	keys := newSchemaKeys([]byte(`{
  "definitions": {
    "job": {
      "oneOf": [
        {"type": "string", "markdownDescription": "A job's name."},
        {"type": "object", "properties": {
          "size": {"markdownDescription": "How big.", "type": "string"},
          "plain": {"description": "Only a description.", "type": "string"},
          "linked": {"$ref": "#/definitions/linked"},
          "bare": {"type": "string"}
        }}
      ]
    },
    "linked": {"markdownDescription": "Described where it's defined."}
  },
  "properties": {
    "jobs": {
      "markdownDescription": "The jobs.",
      "additionalProperties": {"$ref": "#/definitions/job"}
    }
  }
}`))
	assert.Assert(t, keys != nil)

	testCases := []struct {
		name   string
		fields []string
		want   string
	}{
		{name: "a top-level key", fields: []string{"jobs"}, want: "The jobs."},
		{name: "a key in one branch of a oneOf", fields: []string{"jobs", "build", "size"}, want: "How big."},
		{name: "a key with only a description", fields: []string{"jobs", "build", "plain"}, want: "Only a description."},
		{name: "a key described by its $ref", fields: []string{"jobs", "build", "linked"}, want: "Described where it's defined."},
		{name: "a key with no description", fields: []string{"jobs", "build", "bare"}, want: ""},
		{name: "a name the config chooses", fields: []string{"jobs", "build"}, want: ""},
		{name: "a key the schema doesn't have", fields: []string{"jobs", "build", "nope"}, want: ""},
		{name: "no key", fields: nil, want: ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := keys.description(tc.fields)
			assert.Check(t, cmp.Equal(got, tc.want))
		})
	}

	t.Run("the built-in schema describes a step's option", func(t *testing.T) {
		got := SchemaKeyDescription([]string{"jobs", "build", "steps", "0", "run", "command"})
		assert.Check(t, cmp.Equal(got, "Command to run via the shell"))
	})
}

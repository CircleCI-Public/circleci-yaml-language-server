package pipelinevalues

import (
	"slices"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestLookup(t *testing.T) {
	t.Run("A value with a type", func(t *testing.T) {
		value, ok := Lookup("pipeline.number")
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(value.Type, "uint"))
		assert.Check(t, value.Definition != "")
	})

	t.Run("A value without a type is a string", func(t *testing.T) {
		value, ok := Lookup("pipeline.git.branch")
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(value.Type, "string"))
		assert.Check(t, !value.Private)
	})

	t.Run("A replaced value", func(t *testing.T) {
		value, ok := Lookup("pipeline.schedule.name")
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(value.ReplacedBy, "pipeline.trigger.name"))
	})

	t.Run("An unknown value", func(t *testing.T) {
		_, ok := Lookup("pipeline.not_a_value")
		assert.Check(t, !ok)
	})
}

func TestPublic(t *testing.T) {
	public := Public()
	names := []string{}
	for _, value := range public {
		names = append(names, value.Name)
		assert.Check(t, !value.Private, value.Name)
	}

	assert.Check(t, cmp.Contains(names, "pipeline.git.branch"))
	assert.Check(t, !slices.Contains(names, "pipeline.trigger.name"), "private values are left out")
	assert.Check(t, !slices.Contains(names, "pipeline.parameters.*"), "the pipeline parameters are left out")
	assert.Check(t, slices.IsSorted(names))
}

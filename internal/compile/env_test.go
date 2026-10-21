package compile

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/ir"
)

func TestEnv(t *testing.T) {
	env := Env{
		"pipeline.number":       int64(1),
		"pipeline.git.branch":   "main",
		"pipelines.not":         "x",
		"parameters.version":    "1",
		"parameters.extra.args": "-v",
	}

	t.Run("With adds without changing the original", func(t *testing.T) {
		got := env.With(map[string]any{"parameters.version": "2", "parameters.new": true})
		assert.Check(t, cmp.Equal(got["parameters.version"], "2"))
		assert.Check(t, cmp.Equal(got["parameters.new"], true))
		assert.Check(t, cmp.Equal(env["parameters.version"], "1"))
		assert.Check(t, cmp.Len(env, 5))
	})

	t.Run("WithoutPrefix drops a whole prefix", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(env.WithoutPrefix("pipeline"), Env{
			"pipelines.not":         "x",
			"parameters.version":    "1",
			"parameters.extra.args": "-v",
		}))
	})

	t.Run("Only keeps a whole prefix", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(env.Only("pipeline"), Env{
			"pipeline.number":     int64(1),
			"pipeline.git.branch": "main",
		}))
	})

	t.Run("With on nil", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(Env(nil).With(map[string]any{"a": 1}), Env{"a": 1}))
	})
}

func TestError(t *testing.T) {
	err := &Error{
		Message: "Unknown variable(s): parameters.x",
		Loc:     ir.Loc{Source: "orb.yml", Line: 12, Col: 7},
		Frames: []ir.Frame{
			{Kind: "command", Name: "greet", Loc: ir.Loc{Source: "config.yml", Line: 20, Col: 9}},
			{Kind: "job", Name: "build", Loc: ir.Loc{Source: "config.yml", Line: 40, Col: 7}},
		},
	}
	assert.Check(t, cmp.Equal(err.Error(), `Unknown variable(s): parameters.x
  at orb.yml:12:7
  in command 'greet', called from config.yml:20:9
  in job 'build', called from config.yml:40:7`))
	assert.Check(t, cmp.Equal((&Error{Message: "bare"}).Error(), "bare"))
}

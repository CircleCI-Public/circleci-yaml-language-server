package parser

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

// targets maps each alias's name to its target.
func targets(aliases map[string]ast2.Alias) map[string]string {
	res := map[string]string{}
	for name, alias := range aliases {
		res[name] = alias.Target
	}
	return res
}

func TestExecutorAliasesParse(t *testing.T) {
	const config = `version: 2.1
executors:
  plain: orb/e
  quoted: "real"
  number: 5
  real:
    machine:
      image: ubuntu-2404:current
`
	doc, err := ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
	assert.NilError(t, err)
	defer doc.Close()

	t.Run("string entries are aliases", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(targets(doc.Aliases.Executors), map[string]string{
			"plain":  "orb/e",
			"quoted": "real",
		}))
	})

	t.Run("aliases aren't executors", func(t *testing.T) {
		assert.Check(t, cmp.Contains(doc.Executors, "real"))
		assert.Check(t, !doc.DoesExecutorExist("number"))
		_, isExecutor := doc.Executors["plain"]
		assert.Check(t, !isExecutor)
	})

	t.Run("an alias's range is its target's", func(t *testing.T) {
		alias := doc.Aliases.Executors["plain"]
		assert.Check(t, cmp.DeepEqual(alias.TargetRange, protocol.Range{
			Start: protocol.Position{Line: 2, Character: 9},
			End:   protocol.Position{Line: 2, Character: 14},
		}))
	})
}

func TestAliasOrbTarget(t *testing.T) {
	for target, want := range map[string]bool{
		"orb/build":       true,
		"orb/build/extra": false,
		"45m":             false,
		"/build":          false,
		"orb/":            false,
	} {
		_, _, ok := ast2.Alias{Target: target}.OrbTarget()
		assert.Check(t, cmp.Equal(ok, want), "target %q", target)
	}
}

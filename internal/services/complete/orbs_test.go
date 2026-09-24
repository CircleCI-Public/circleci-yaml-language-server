package complete

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

// orbCompletionHandler is a handler completing against the fake, with a cache
// of its own.
func orbCompletionHandler(fake *fakes.CircleCI) *CompletionHandler {
	settings := testHelpers.SettingsForHost(fake.URL())

	return &CompletionHandler{
		Doc:     yamlparser.YamlDocument{Context: settings},
		Cache:   cache.New(),
		Context: settings,
	}
}

func TestGetOrbNameCompletions(t *testing.T) {
	t.Run("suggests each orb at its newest version", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		fake.AddOrbPackage("orb-node", "ns-circleci", "circleci", "node", false, true)
		fake.AddOrbVersion("ver-node-1", "orb-node", "circleci/node", "1.0.0", "", "")
		fake.AddOrbVersion("ver-node-2", "orb-node", "circleci/node", "7.2.1", "", "")

		completions, err := orbCompletionHandler(fake).getOrbNameCompletions("circleci/")
		assert.NilError(t, err)

		assert.Check(t, cmp.DeepEqual(completions, []string{
			"circleci/go@4.0.0",
			"circleci/node@7.2.1",
		}))
	})

	// An orb with nothing published has no reference to complete to, and used
	// to contribute an empty string to the completion list.
	t.Run("skips an orb with no published versions", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		fake.AddOrbPackage("orb-empty", "ns-circleci", "circleci", "empty", false, true)
		fake.AddOrbPackage("orb-go", "ns-circleci", "circleci", "go", false, true)
		fake.AddOrbVersion("ver-go", "orb-go", "circleci/go", "1.0.0", "", "")

		completions, err := orbCompletionHandler(fake).getOrbNameCompletions("circleci/")
		assert.NilError(t, err)

		assert.Check(t, cmp.DeepEqual(completions, []string{"circleci/go@1.0.0"}))
	})

	t.Run("suggests nothing for an unknown namespace", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)

		completions, err := orbCompletionHandler(fake).getOrbNameCompletions("nope/")
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(completions, 0))
	})
}

func TestGetOrbVersionCompletions(t *testing.T) {
	t.Run("suggests an orb's versions newest first", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		completions, err := orbCompletionHandler(fake).getOrbVersionCompletions("circleci/go@")
		assert.NilError(t, err)

		assert.Check(t, cmp.DeepEqual(completions, []string{
			"4.0.0", "1.12.0", "1.7.3", "1.7.1", "1.7.0", "0.1.0",
		}))
	})

	t.Run("reports an unknown orb", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		_, err := orbCompletionHandler(fake).getOrbVersionCompletions("circleci/nope")
		assert.Check(t, cmp.ErrorContains(err, "no orb named circleci/nope"))
	})
}

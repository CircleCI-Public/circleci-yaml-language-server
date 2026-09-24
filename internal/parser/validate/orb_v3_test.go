package validate

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

// isolateOrbSources points the temporary directory, which fetched orb
// sources are written under, somewhere the test cleans up.
func isolateOrbSources(t *testing.T) {
	t.Helper()

	t.Setenv("TMPDIR", t.TempDir())
}

// orbDiagnostics validates a config against the fake and returns the
// diagnostics its orbs produced.
func orbDiagnostics(t *testing.T, fake *fakes.CircleCI, yamlContent string) []protocol.Diagnostic {
	t.Helper()

	lsContext := testHelpers.SettingsForHost(fake.URL())
	doc, err := parser.ParseFromContent([]byte(yamlContent), lsContext, uri.File("config.yml"), protocol.Position{})
	assert.NilError(t, err)

	diagnostics := []protocol.Diagnostic{}
	val := Validate{
		APIs:        ValidateAPIs{DockerHub: DockerHubMock{}},
		Diagnostics: &diagnostics,
		Cache:       cache.New(),
		Doc:         doc,
		Context:     lsContext,
	}

	val.ValidateOrbs()

	return *val.Diagnostics
}

// These exercise the whole orb path end to end — resolve the reference, fetch
// the source, parse it, compare versions — over the V3 API.
func TestOrbValidationOverV3(t *testing.T) {
	// Each subtest uses its own orb name because parser memoises orb existence
	// in a package-level map that a test cannot reach.
	t.Run("accepts an orb at its latest version", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-latest", "ns-acme", "acme", "at-latest", false, true)
		fake.AddOrbVersion("ver-latest-1", "orb-latest", "acme/at-latest", "1.0.0", orbSource, "")

		diagnostics := orbDiagnostics(t, fake, `version: 2.1

orbs:
  thing: acme/at-latest@1.0.0

jobs:
  build:
    executor: thing/default
    steps:
      - thing/greet

workflows:
  main:
    jobs:
      - build
`)

		messages := diagnosticMessages(&diagnostics)
		assert.Check(t, cmp.DeepEqual(messages, []string{}))
	})

	// @N and @N.M are prefix ranges. Comparing only as deep as the pin
	// means @0 is not 0.0.0, but @6 still notices @7 (PIPE-9822).
	t.Run("does not warn that a major-only pin is behind on patch or minor", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-major", "ns-acme", "acme", "major-pin", false, true)
		fake.AddOrbVersion("ver-major-1", "orb-major", "acme/major-pin", "0.1.11", orbSource, "")
		fake.AddOrbVersion("ver-major-2", "orb-major", "acme/major-pin", "0.2.0", orbSource, "")

		diagnostics := orbDiagnostics(t, fake, `version: 2.1

orbs:
  thing: acme/major-pin@0

jobs:
  build:
    executor: thing/default
    steps:
      - thing/greet

workflows:
  main:
    jobs:
      - build
`)

		messages := diagnosticMessages(&diagnostics)
		assert.Check(t, cmp.DeepEqual(messages, []string{}))
	})

	t.Run("reports a newer major for a major-only pin", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-stale-major", "ns-acme", "acme", "stale-major", false, true)
		fake.AddOrbVersion("ver-stale-major-1", "orb-stale-major", "acme/stale-major", "1.9.0", orbSource, "")
		fake.AddOrbVersion("ver-stale-major-2", "orb-stale-major", "acme/stale-major", "2.0.0", orbSource, "")

		diagnostics := orbDiagnostics(t, fake, `version: 2.1

orbs:
  thing: acme/stale-major@1

jobs:
  build:
    executor: thing/default
    steps:
      - thing/greet

workflows:
  main:
    jobs:
      - build
`)

		assert.Assert(t, cmp.Len(diagnostics, 1))
		assert.Check(t, cmp.Contains(diagnostics[0].Message, "newer major"))
		assert.Check(t, cmp.Contains(diagnostics[0].Message, "2.0.0"))
		assert.Check(t, cmp.Equal(diagnostics[0].Severity, protocol.DiagnosticSeverityInformation))
	})

	t.Run("reports a newer minor for a major.minor pin", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-stale-minor", "ns-acme", "acme", "stale-minor", false, true)
		fake.AddOrbVersion("ver-stale-minor-1", "orb-stale-minor", "acme/stale-minor", "1.2.0", orbSource, "")
		fake.AddOrbVersion("ver-stale-minor-2", "orb-stale-minor", "acme/stale-minor", "1.2.3", orbSource, "")
		fake.AddOrbVersion("ver-stale-minor-3", "orb-stale-minor", "acme/stale-minor", "1.3.0", orbSource, "")

		diagnostics := orbDiagnostics(t, fake, `version: 2.1

orbs:
  thing: acme/stale-minor@1.2

jobs:
  build:
    executor: thing/default
    steps:
      - thing/greet

workflows:
  main:
    jobs:
      - build
`)

		assert.Assert(t, cmp.Len(diagnostics, 1))
		assert.Check(t, cmp.Contains(diagnostics[0].Message, "newer minor"))
		assert.Check(t, cmp.Contains(diagnostics[0].Message, "1.3.0"))
		assert.Check(t, cmp.Equal(diagnostics[0].Severity, protocol.DiagnosticSeverityInformation))
	})
	t.Run("reports a newer version of an out-of-date orb", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-stale", "ns-acme", "acme", "stale", false, true)
		fake.AddOrbVersion("ver-stale-1", "orb-stale", "acme/stale", "1.0.0", orbSource, "")
		fake.AddOrbVersion("ver-stale-2", "orb-stale", "acme/stale", "2.0.0", orbSource, "")

		diagnostics := orbDiagnostics(t, fake, `version: 2.1

orbs:
  thing: acme/stale@1.0.0

jobs:
  build:
    executor: thing/default
    steps:
      - thing/greet

workflows:
  main:
    jobs:
      - build
`)

		assert.Assert(t, cmp.Len(diagnostics, 1))
		assert.Check(t, cmp.Contains(diagnostics[0].Message, "2.0.0"))
	})

	t.Run("flags an orb that does not exist", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")

		diagnostics := orbDiagnostics(t, fake, `version: 2.1

orbs:
  thing: acme/absent@1.0.0

jobs:
  build:
    executor: thing/default
    steps:
      - thing/greet

workflows:
  main:
    jobs:
      - build
`)

		assert.Assert(t, cmp.Len(diagnostics, 1))
		assert.Check(t, cmp.Contains(diagnostics[0].Message, "does not exist or is private"))
	})

	t.Run("flags a version that does not exist", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-badver", "ns-acme", "acme", "badver", false, true)
		fake.AddOrbVersion("ver-badver-1", "orb-badver", "acme/badver", "1.0.0", orbSource, "")

		diagnostics := orbDiagnostics(t, fake, `version: 2.1

orbs:
  thing: acme/badver@9.9.9

jobs:
  build:
    executor: thing/default
    steps:
      - thing/greet

workflows:
  main:
    jobs:
      - build
`)

		assert.Assert(t, cmp.Len(diagnostics, 1))
		assert.Check(t, cmp.Contains(diagnostics[0].Message, "Unknown version 9.9.9"))
	})

	t.Run("resolves an orb pinned to a partial version", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-partial", "ns-acme", "acme", "partial", false, true)
		fake.AddOrbVersion("ver-partial-1", "orb-partial", "acme/partial", "1.2.0", orbSource, "")
		fake.AddOrbVersion("ver-partial-2", "orb-partial", "acme/partial", "1.2.3", orbSource, "")

		diagnostics := orbDiagnostics(t, fake, `version: 2.1

orbs:
  thing: acme/partial@1.2

jobs:
  build:
    executor: thing/default
    steps:
      - thing/greet

workflows:
  main:
    jobs:
      - build
`)

		messages := diagnosticMessages(&diagnostics)
		assert.Check(t, cmp.DeepEqual(messages, []string{}))
	})

	// Without a token the language server still resolves public orbs, so an
	// anonymous run must produce the same diagnostics as an authenticated one.
	t.Run("resolves a public orb without a token", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-anon", "ns-acme", "acme", "anon", false, true)
		fake.AddOrbVersion("ver-anon-1", "orb-anon", "acme/anon", "1.0.0", orbSource, "")

		lsContext := testHelpers.SettingsForHost(fake.URL())
		lsContext.Api.Token = ""

		yamlContent := `version: 2.1

orbs:
  thing: acme/anon@1.0.0

jobs:
  build:
    executor: thing/default
    steps:
      - thing/greet

workflows:
  main:
    jobs:
      - build
`

		doc, err := parser.ParseFromContent([]byte(yamlContent), lsContext, uri.File("config.yml"), protocol.Position{})
		assert.NilError(t, err)

		diagnostics := []protocol.Diagnostic{}
		val := Validate{
			APIs:        ValidateAPIs{DockerHub: DockerHubMock{}},
			Diagnostics: &diagnostics,
			Cache:       cache.New(),
			Doc:         doc,
			Context:     lsContext,
		}
		val.ValidateOrbs()

		messages := diagnosticMessages(val.Diagnostics)
		assert.Check(t, cmp.DeepEqual(messages, []string{}))

		requests := fake.Requests()
		assert.Assert(t, len(requests) != 0)
		for _, request := range requests {
			assert.Check(t, cmp.Equal(request.Authorization, ""), "path %s", request.Path)
		}
	})
}

// orbSource is a minimal orb exposing one executor and one command, enough for
// a config to reference it without tripping other validation.
const orbSource = `version: 2.1

description: A test orb

executors:
  default:
    docker:
      - image: cimg/base:current

commands:
  greet:
    steps:
      - run: echo hello
`

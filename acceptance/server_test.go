package acceptance

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/lspclient"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/runner"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/workspace"
)

const (
	// orgID is the organization the acme/rocket project belongs to. Contexts
	// and environment variables are looked up by it, not by the project.
	orgID = "org-acme"

	// testToken is the token a test presents, set the way the extension sets
	// it: over executeCommand, after the handshake.
	testToken = "test-token"
)

// Routes, as the fake records them, so an assertion can say which of the
// server's reads happened.
//
// runnerPath belongs to the runner service rather than to the CircleCI V3 API,
// despite sharing its prefix; the server reaches it on a runner. subdomain,
// which is why the tests override that host.
const (
	projectPath    = "/api/v2/project/gh/acme/rocket"
	envVarPath     = "/api/v2/project/gh/acme/rocket/envvar"
	contextPath    = "/api/v2/context"
	orbPackagePath = "/api/v3/orb/packages"
	runnerPath     = "/api/v3/runner/resource"
)

// validConfig is a config with nothing wrong with it, using a machine executor
// rather than a Docker image: Docker images are checked against the real Docker
// Hub, which has no seam to point at a fake yet.
const validConfig = `version: 2.1

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo building

workflows:
  main:
    jobs:
      - build:
          context:
            - deploy
`

// unknownContextConfig names a context the organization does not have, which
// the server can only know by having read the organization's contexts.
const unknownContextConfig = `version: 2.1

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo building

workflows:
  main:
    jobs:
      - build:
          context:
            - nope
`

// orbConfig uses a public orb and nothing that needs a token.
const orbConfig = `version: 2.1

orbs:
  go: circleci/go@1.7.1

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo building

workflows:
  main:
    jobs:
      - build
`

// linkedProjectFake is a CircleCI that knows the acme/rocket project, the
// contexts and variables of its organization, the machine catalog, the runner
// resource classes of the namespace, and the circleci/go orb.
func linkedProjectFake(t *testing.T) *fakes.CircleCI {
	t.Helper()

	return linkedProjectFakeIn(t, orgID)
}

// linkedProjectFakeIn is linkedProjectFake with the project in an organization
// of its own, for a case that needs two hosts to disagree about it.
func linkedProjectFakeIn(t *testing.T, orgID string) *fakes.CircleCI {
	t.Helper()

	fake := fakes.NewCircleCI(t)

	fake.SetUser("user-jane", "jane", "Jane Doe")
	fake.AddProject(workspace.DefaultSlug, "proj-rocket", orgID, "gh/acme")
	fake.AddProjectEnvVar(workspace.DefaultSlug, "AWS_REGION", "")
	fake.AddContext(orgID, "ctx-deploy", "acme/deploy")
	fake.AddContextEnvVar("ctx-deploy", "DEPLOY_KEY")
	fake.AddRunnerResourceClass("acme", "acme/linux-arm", "ARM builders")
	fake.SetMachineOfferings(fakes.MachineOfferings{
		Linux: map[string][]string{"medium": {"ubuntu-2404:current"}},
	})
	fake.SeedGoOrb()

	return fake
}

// session is a running server, the client driving it, and the workspace it has
// been pointed at.
type session struct {
	client    *lspclient.Client
	server    *runner.Server
	workspace *workspace.Workspace
}

// start compiles nothing and configures everything: it runs the server over
// stdio, hands it the fake's address and a token the way the extension does,
// and leaves the document unopened so a test can time that itself.
//
// An empty token leaves the server unauthenticated, which is how it serves
// someone who has not logged in.
func start(t *testing.T, fake *fakes.CircleCI, config, token string) *session {
	t.Helper()

	return startIn(t, fake, workspace.New(t, config), token)
}

// startIn is start in a workspace a test has made itself, for a case about
// the repository the config lives in.
func startIn(t *testing.T, fake *fakes.CircleCI, project *workspace.Workspace, token string) *session {
	t.Helper()

	server := runner.StartStdio(t, serverBinary, "CIRCLECI_RUNNER_HOST="+fake.URL())
	client := lspclient.New(t, context.Background(), server.Stream())

	_, err := client.Initialize(project.RootURI())
	assert.NilError(t, err)

	err = client.ExecuteCommand("setSelfHostedUrl", fake.URL())
	assert.NilError(t, err)

	if token != "" {
		err = client.ExecuteCommand("setToken", token)
		assert.NilError(t, err)
	}

	return &session{client: client, server: server, workspace: project}
}

// open opens the workspace's config and returns the diagnostics the server
// publishes for it.
func (s *session) open(t *testing.T, config string) []string {
	t.Helper()

	err := s.client.DidOpen(s.workspace.URI(), config)
	assert.NilError(t, err)

	diagnostics, err := s.client.WaitForDiagnostics(s.workspace.URI())
	assert.NilError(t, err)

	return messages(diagnostics)
}

func TestInitialize(t *testing.T) {
	fake := linkedProjectFake(t)
	project := workspace.New(t, validConfig)
	server := runner.StartStdio(t, serverBinary)
	client := lspclient.New(t, context.Background(), server.Stream())

	result, err := client.Initialize(project.RootURI())
	assert.NilError(t, err)
	assert.Assert(t, result != nil)

	// The extension turns these on for the features it offers, so a server
	// that stopped advertising one would silently lose it.
	t.Run("advertises the capabilities a client needs", func(t *testing.T) {
		assert.Check(t, result.Capabilities.CompletionProvider != nil, "completion")
		assert.Check(t, result.Capabilities.HoverProvider != nil, "hover")
		assert.Check(t, result.Capabilities.DefinitionProvider != nil, "definition")
		assert.Check(t, result.Capabilities.CodeActionProvider != nil, "code actions")
		assert.Check(t, result.Capabilities.ExecuteCommandProvider != nil, "commands")
		assert.Check(t, result.Capabilities.SemanticTokensProvider != nil, "semantic tokens")
	})

	// Nothing has been opened, so nothing should have been read from the API.
	t.Run("reads nothing from the API", func(t *testing.T) {
		requests := fake.Requests()
		assert.Check(t, cmp.Len(requests, 0))
	})
}

func TestOpenLinkedProject(t *testing.T) {
	fake := linkedProjectFake(t)
	session := start(t, fake, validConfig, testToken)

	diagnostics := session.open(t, validConfig)

	t.Run("reports nothing wrong with a valid config", func(t *testing.T) {
		assert.Check(t, cmp.Len(diagnostics, 0))
	})

	// The chain is project, then the organization's contexts, then the
	// project's own variables — each keyed on what the one before it reported.
	t.Run("resolves the project, its contexts and its variables", func(t *testing.T) {
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, projectPath), 1))
		assert.Check(t, fake.RequestCount(http.MethodGet, contextPath) > 0, "contexts must be read")
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, envVarPath), 1))
	})

	t.Run("looks the contexts up by the organization the project reported", func(t *testing.T) {
		for _, request := range fake.Requests() {
			if request.Path == contextPath {
				assert.Check(t, cmp.Equal(request.Query["owner-id"], orgID))
			}
		}
	})

	t.Run("presents the token it was given", func(t *testing.T) {
		requests := fake.Requests()
		assert.Assert(t, len(requests) != 0)

		for _, request := range requests {
			assert.Check(t, cmp.Equal(request.CircleToken, testToken), "%s %s", request.Method, request.Path)
		}
	})

	// The runner API is a different host in production, so this only works
	// because it is overridable — and without it the server would spend every
	// open waiting on a name that does not resolve.
	t.Run("reads the organization's runner resource classes", func(t *testing.T) {
		eventually(t, "the runner API to be read", func() bool {
			return fake.RequestCount(http.MethodGet, runnerPath) > 0
		})
	})
}

// `git clone https://…` leaves a remote ending in .git, which has to resolve to
// the same project as one without it.
func TestOpenProjectClonedOverHttps(t *testing.T) {
	fake := linkedProjectFake(t)
	project := workspace.NewWithRemote(t, validConfig, workspace.DefaultRemote+".git")
	session := startIn(t, fake, project, testToken)

	session.open(t, validConfig)

	t.Run("resolves the project, its contexts and its variables", func(t *testing.T) {
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, projectPath), 1))
		assert.Check(t, fake.RequestCount(http.MethodGet, contextPath) > 0, "contexts must be read")
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, envVarPath), 1))
	})
}

func TestUnknownContextIsADiagnostic(t *testing.T) {
	fake := linkedProjectFake(t)
	session := start(t, fake, unknownContextConfig, testToken)

	diagnostics := session.open(t, unknownContextConfig)

	assert.Check(t, cmp.Contains(diagnostics, "Context nope does not exist"))
}

func TestSetToken(t *testing.T) {
	fake := linkedProjectFake(t)
	session := start(t, fake, validConfig, "")

	// Opened unauthenticated: the routes that need a token are not called.
	session.open(t, validConfig)
	assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, projectPath), 0))

	err := session.client.ExecuteCommand("setToken", testToken)
	assert.NilError(t, err)

	// Signing in revalidates what is already open, which is the only way the
	// diagnostics of an open document can start depending on the API.
	t.Run("revalidates the open document against the API", func(t *testing.T) {
		_, err := session.client.WaitForDiagnostics(session.workspace.URI())
		assert.NilError(t, err)

		eventually(t, "the project to be read", func() bool {
			return fake.RequestCount(http.MethodGet, projectPath) > 0
		})

		requests := fake.Requests()
		assert.Check(t, cmp.Equal(requests[len(requests)-1].CircleToken, testToken))
	})
}

func TestSetSelfHostedUrl(t *testing.T) {
	// The same project lives in a different organization on each host, which
	// is what a self-hosted install of a cloud project looks like.
	const secondOrgID = "org-acme-server"

	first := linkedProjectFake(t)
	second := linkedProjectFakeIn(t, secondOrgID)

	session := start(t, first, validConfig, testToken)
	session.open(t, validConfig)

	assert.Assert(t, first.RequestCount(http.MethodGet, projectPath) > 0)

	// Opening the document also reads the runner resource classes, in the
	// background, from the runner host start pinned to the first fake. Wait for
	// that read so it is not counted as the first host being read after the
	// switch.
	eventually(t, "the runner API to be read", func() bool {
		return first.RequestCount(http.MethodGet, runnerPath) > 0
	})

	err := session.client.ExecuteCommand("setSelfHostedUrl", second.URL())
	assert.NilError(t, err)

	readFromFirst := len(first.Requests())

	t.Run("reads the new host", func(t *testing.T) {
		_, err := session.client.WaitForDiagnostics(session.workspace.URI())
		assert.NilError(t, err)

		eventually(t, "the second host to be read", func() bool {
			return second.RequestCount(http.MethodGet, contextPath) > 0
		})
	})

	t.Run("stops reading the old one", func(t *testing.T) {
		err := session.client.DidChange(session.workspace.URI(), 2, validConfig)
		assert.NilError(t, err)

		_, err = session.client.WaitForDiagnostics(session.workspace.URI())
		assert.NilError(t, err)

		assert.Check(t, cmp.Equal(len(first.Requests()), readFromFirst),
			"nothing more should be read from the host that was replaced")
	})

	// The project resolved on the old host is forgotten with the rest of its
	// data, so the new host's contexts are looked up under the organization
	// the new host reports.
	t.Run("re-resolves the project on the new host", func(t *testing.T) {
		assert.Check(t, second.RequestCount(http.MethodGet, projectPath) > 0, "the project must be read again")

		for _, request := range second.Requests() {
			if request.Path == contextPath {
				assert.Check(t, cmp.Equal(request.Query["owner-id"], secondOrgID),
					"the organization id comes from the new host")
			}
		}
	})
}

func TestUnauthenticatedOpen(t *testing.T) {
	fake := linkedProjectFake(t)
	session := start(t, fake, orbConfig, "")

	diagnostics := session.open(t, orbConfig)

	// A public orb is readable without a token, and the language server serves
	// people who have not logged in.
	t.Run("resolves a public orb", func(t *testing.T) {
		assert.Check(t, fake.RequestCount(http.MethodGet, orbPackagePath) > 0, "the orb must be resolved")

		// The versions in the upgrade hint can only have come from the fake,
		// so this is the orb resolving end to end rather than a request count.
		said := strings.Join(diagnostics, "\n")
		assert.Check(t, cmp.Contains(said, "A newer patched version exists"))
		assert.Check(t, cmp.Contains(said, "4.0.0"), "the hint names the latest version the fake has")
	})

	t.Run("calls nothing that needs a token", func(t *testing.T) {
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, projectPath), 0))
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, contextPath), 0))
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, envVarPath), 0))
	})
}

func TestApiOutage(t *testing.T) {
	t.Run("still validates when the API fails", func(t *testing.T) {
		fake := linkedProjectFake(t)
		fake.SetStatus("GET "+projectPath, http.StatusInternalServerError)
		fake.SetStatus("GET "+contextPath, http.StatusInternalServerError)

		session := start(t, fake, validConfig, testToken)

		diagnostics := session.open(t, validConfig)
		assert.Check(t, cmp.Len(diagnostics, 0), "a valid config is still valid without the API")
	})

	t.Run("still validates when the API has gone", func(t *testing.T) {
		fake := linkedProjectFake(t)
		session := start(t, fake, unknownContextConfig, testToken)
		fake.Close()

		// Without the contexts, the context that does not exist cannot be
		// reported — but the document still gets diagnostics rather than
		// nothing, and the server stays up.
		diagnostics := session.open(t, unknownContextConfig)
		assert.Check(t, cmp.Len(diagnostics, 0))

		_, err := session.client.Completion(session.workspace.URI(), position(1, 0))
		assert.NilError(t, err, "the server must still answer requests")
	})
}

func TestSocketTransport(t *testing.T) {
	fake := linkedProjectFake(t)
	project := workspace.New(t, validConfig)

	// The extension starts the server this way and waits for the line it
	// prints before connecting, so the line is part of the contract.
	server := runner.StartSocket(t, serverBinary, "CIRCLECI_RUNNER_HOST="+fake.URL())
	assert.Check(t, server.Port() != 0, "the server must report the port it bound")

	client := lspclient.New(t, context.Background(), server.Stream())

	result, err := client.Initialize(project.RootURI())
	assert.NilError(t, err)
	assert.Assert(t, result != nil)
	assert.Check(t, result.Capabilities.CompletionProvider != nil)

	t.Run("serves a document over the socket", func(t *testing.T) {
		err := client.ExecuteCommand("setSelfHostedUrl", fake.URL())
		assert.NilError(t, err)

		err = client.DidOpen(project.URI(), validConfig)
		assert.NilError(t, err)

		diagnostics, err := client.WaitForDiagnostics(project.URI())
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(diagnostics, 0))
	})
}

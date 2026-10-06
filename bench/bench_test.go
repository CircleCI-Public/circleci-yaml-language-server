// Package bench times the language server from the outside, the way gopls's
// benchmarks time gopls: the real binary, run as a subprocess and driven over
// the protocol an editor speaks, against generated configs of fixed sizes and
// a fake CircleCI in the benchmark's own process.
//
// Compare two runs with benchstat; see HACKING.md.
package bench

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/compiler"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/lspclient"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/runner"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/workspace"
)

// serverBinary is the compiled language server every benchmark runs. TestMain
// builds it, but only for a run that has benchmarks to time: `task test` runs
// this package too, and has nothing here to wait for.
var serverBinary string

func TestMain(m *testing.M) {
	flag.Parse()

	if flag.Lookup("test.bench").Value.String() == "" {
		os.Exit(m.Run())
	}

	status, err := runBenchmarks(m)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	os.Exit(status)
}

func runBenchmarks(m *testing.M) (int, error) {
	binaries := compiler.NewParallel(1)
	defer binaries.Cleanup()

	binaries.Add(compiler.Work{
		Result: &serverBinary,
		Name:   "server",
		Target: "..",
		Source: "./cmd/server",
	})

	if err := binaries.Run(context.Background()); err != nil {
		return 0, err
	}

	return m.Run(), nil
}

const (
	orgID     = "org-acme"
	testToken = "test-token"
)

// linkedProject is a CircleCI that knows the project a workspace resolves to,
// its organization and the machine image the fixtures run on, so that opening
// a fixture does the work it would for a signed-in user and reports nothing
// wrong.
func linkedProject(b *testing.B) *fakes.CircleCI {
	b.Helper()

	fake := fakes.NewCircleCI(b)
	fake.SetUser("user-jane", "jane", "Jane Doe")
	fake.AddProject(workspace.DefaultSlug, "proj-rocket", orgID, "gh/acme")
	fake.AddOrg("gh/acme", orgID)
	fake.SetMachineOfferings(fakes.MachineOfferings{
		Linux: map[string][]string{"medium": {"ubuntu-2404:current"}},
	})

	return fake
}

// session is a running, initialized server and the client driving it.
type session struct {
	client *lspclient.Client
	server *runner.Server
}

// start runs a server, initializes it as the extension does with options of
// its own on top, and signs it in to fake.
func start(b *testing.B, fake *fakes.CircleCI, project *workspace.Workspace, options map[string]any) *session {
	b.Helper()

	dockerHub := fakes.NewDockerHub(b)
	server := runner.StartStdio(b, serverBinary, "LSP_DOCKER_HUB_URL="+dockerHub.URL())
	client := lspclient.New(b, b.Context(), server.Stream())

	initOptions := map[string]any{"isCciExtension": true}
	for key, value := range options {
		initOptions[key] = value
	}

	_, err := client.InitializeWithOptions(project.RootURI(), initOptions)
	assert.NilError(b, err)
	assert.NilError(b, client.ExecuteCommand("setSelfHostedUrl", fake.URL()))
	assert.NilError(b, client.ExecuteCommand("setToken", testToken))

	return &session{client: client, server: server}
}

// quiet is how long a server has to publish nothing before it counts as
// settled: twice the second it waits after reading from the API before it
// checks an open document again.
const quiet = 2 * time.Second

// open opens the fixture, checks that the server finds nothing wrong with it,
// and waits for the server to settle, so that the work opening started does
// not land in what a benchmark times next.
func (s *session) open(b *testing.B, project *workspace.Workspace, f fixture) {
	b.Helper()

	assert.NilError(b, s.client.DidOpen(project.URI(), f.config))

	diagnostics, err := s.client.WaitForDiagnostics(project.URI())
	assert.NilError(b, err)
	assert.Check(b, cmp.Len(diagnostics, 0), "the %s fixture should be valid", f.name)

	for {
		if _, err := s.client.WaitForDiagnosticsWithin(project.URI(), quiet); err != nil {
			return
		}
	}
}

// BenchmarkStartup times starting a server and the handshake that leaves it
// ready for a document: what opening an editor costs before any config is
// open.
func BenchmarkStartup(b *testing.B) {
	project := workspace.New(b, small.config)

	for range b.N {
		server := runner.StartStdio(b, serverBinary)
		client := lspclient.New(b, b.Context(), server.Stream())

		_, err := client.InitializeWithOptions(project.RootURI(), map[string]any{"isCciExtension": true})
		assert.NilError(b, err)

		b.StopTimer()
		server.Stop()
		b.StartTimer()
	}
}

// BenchmarkOpen times opening a config in a server that has just started, up
// to the first diagnostics it publishes: how long a user waits to see what is
// wrong with a config they have just opened.
func BenchmarkOpen(b *testing.B) {
	for _, f := range fixtures {
		b.Run(f.name, func(b *testing.B) {
			fake := linkedProject(b)
			project := workspace.New(b, f.config)

			for range b.N {
				b.StopTimer()
				s := start(b, fake, project, nil)
				b.StartTimer()

				assert.NilError(b, s.client.DidOpen(project.URI(), f.config))
				_, err := s.client.WaitForDiagnostics(project.URI())
				assert.NilError(b, err)

				b.StopTimer()
				s.server.Stop()
				b.StartTimer()
			}
		})
	}
}

// BenchmarkChange times checking a config again after an edit, from the
// change to the diagnostics for it, with the server's wait for typing to stop
// turned off: what every pause in typing costs.
func BenchmarkChange(b *testing.B) {
	for _, f := range fixtures {
		b.Run(f.name, func(b *testing.B) {
			fake := linkedProject(b)
			project := workspace.New(b, f.config)
			s := start(b, fake, project, map[string]any{"editDebounceMs": 0})
			s.open(b, project, f)

			// Each edit adds a comment of its own, so that every version has
			// content the server has not checked before.
			version := int32(1)
			for b.Loop() {
				version++
				edited := fmt.Sprintf("%s# edit %d\n", f.config, version)

				assert.NilError(b, s.client.DidChange(project.URI(), version, edited))
				_, err := s.client.WaitForDiagnostics(project.URI())
				assert.NilError(b, err)
			}
		})
	}
}

// BenchmarkHover times hovering over a step that runs one of the config's own
// commands.
func BenchmarkHover(b *testing.B) {
	for _, f := range fixtures {
		b.Run(f.name, func(b *testing.B) {
			fake := linkedProject(b)
			project := workspace.New(b, f.config)
			s := start(b, fake, project, nil)
			s.open(b, project, f)

			hover, err := s.client.Hover(project.URI(), f.hover)
			assert.NilError(b, err)
			markup, ok := hover.Contents.(*protocol.MarkupContent)
			assert.Assert(b, ok, "hover contents are %T, not markup", hover.Contents)
			assert.Check(b, cmp.Contains(markup.Value, "Set up the tools"))

			for b.Loop() {
				_, err := s.client.Hover(project.URI(), f.hover)
				assert.NilError(b, err)
			}
		})
	}
}

// BenchmarkCompletion times completing a job a workflow requires, where every
// job in the config is offered.
func BenchmarkCompletion(b *testing.B) {
	for _, f := range fixtures {
		b.Run(f.name, func(b *testing.B) {
			fake := linkedProject(b)
			project := workspace.New(b, f.config)
			s := start(b, fake, project, nil)
			s.open(b, project, f)

			list, err := s.client.Completion(project.URI(), f.completion)
			assert.NilError(b, err)
			assert.Check(b, cmp.Equal(len(list.Items), f.jobs))

			for b.Loop() {
				_, err := s.client.Completion(project.URI(), f.completion)
				assert.NilError(b, err)
			}
		})
	}
}

// BenchmarkCompletionNewStep times completing a step just started with "- ",
// where the steps a job can run are offered.
func BenchmarkCompletionNewStep(b *testing.B) {
	for _, f := range fixtures {
		b.Run(f.name, func(b *testing.B) {
			fake := linkedProject(b)
			project := workspace.New(b, f.config)
			s := start(b, fake, project, map[string]any{"editDebounceMs": 0})
			s.open(b, project, f)

			assert.NilError(b, s.client.DidChange(project.URI(), 2, f.withNewStep))
			_, err := s.client.WaitForDiagnostics(project.URI())
			assert.NilError(b, err)

			list, err := s.client.Completion(project.URI(), f.newStep)
			assert.NilError(b, err)
			labels := make([]string, 0, len(list.Items))
			for _, item := range list.Items {
				labels = append(labels, item.Label)
			}
			assert.Check(b, cmp.Contains(labels, "checkout"))
			assert.Check(b, cmp.Contains(labels, "setup-0"))

			for b.Loop() {
				_, err := s.client.Completion(project.URI(), f.newStep)
				assert.NilError(b, err)
			}
		})
	}
}

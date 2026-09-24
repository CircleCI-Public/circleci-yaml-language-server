package languageserver

import (
	"context"
	"fmt"
	"net"
	"testing"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/lspclient"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/workspace"
)

// concurrentConfig reaches the orb registry, the project and the
// organization's contexts, so that validating it reads and writes most of what
// the server caches.
const concurrentConfig = `version: 2.1

orbs:
  go: circleci/go@1.7.1

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    environment:
      GREETING: hello
    steps:
      - checkout
      - run: echo $GREETING

workflows:
  main:
    jobs:
      - build:
          context:
            - deploy
`

// serveInProcess runs a session on this process, so that the race detector
// sees both the server and the requests made of it.
func serveInProcess(t *testing.T, hostURL string) *lspclient.Client {
	t.Helper()

	serverEnd, clientEnd := net.Pipe()
	server := JSONRPCServer{
		ctx:       context.Background(),
		lsContext: &session.Settings{Api: circleci.Config{HostUrl: hostURL}},
	}

	var serving errgroup.Group
	serving.Go(func() error {
		return server.serve(jsonrpc2.NewStream(serverEnd))
	})
	t.Cleanup(func() {
		_ = clientEnd.Close()
		// The session ends because the client went, which it reports as an
		// error; all that matters here is that it has ended.
		_ = serving.Wait()
	})

	return lspclient.New(t, context.Background(), clientEnd)
}

// Queries run alongside the messages after them, so edits, settings changes
// and the validation they start all happen while queries are reading the same
// documents and caches. The race detector is what checks this.
func TestConcurrentSession(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	fake.SetUser("user-jane", "jane", "Jane Doe")
	fake.AddProject(workspace.DefaultSlug, "proj-rocket", "org-acme", "gh/acme")
	fake.AddProjectEnvVar(workspace.DefaultSlug, "AWS_REGION", "")
	fake.AddContext("org-acme", "ctx-deploy", "acme/deploy")
	fake.AddContextEnvVar("ctx-deploy", "DEPLOY_KEY")
	fake.SeedGoOrb()

	ws := workspace.New(t, concurrentConfig)
	doc := ws.URI()
	client := serveInProcess(t, fake.URL())

	t.Run("start the session", func(t *testing.T) {
		_, err := client.Initialize(ws.RootURI())
		assert.NilError(t, err)
		assert.NilError(t, client.ExecuteCommand("setSelfHostedUrl", fake.URL()))
		assert.NilError(t, client.ExecuteCommand("setToken", "token-1"))
		assert.NilError(t, client.DidOpen(doc, concurrentConfig))
		_, err = client.WaitForDiagnostics(doc)
		assert.NilError(t, err)
	})

	t.Run("query while editing and changing settings", func(t *testing.T) {
		var group errgroup.Group

		group.Go(func() error {
			for version := int32(2); version <= 40; version++ {
				edited := concurrentConfig + fmt.Sprintf("# edit %d\n", version)
				if err := client.DidChange(doc, version, edited); err != nil {
					return err
				}
			}
			return nil
		})

		group.Go(func() error {
			for i := range 6 {
				if err := client.ExecuteCommand("setToken", fmt.Sprintf("token-%d", i%2)); err != nil {
					return err
				}
				if err := client.ExecuteCommand("setSelfHostedUrl", fake.URL()); err != nil {
					return err
				}
			}
			return nil
		})

		for range 4 {
			group.Go(func() error {
				for range 10 {
					if err := queryEverything(client, doc); err != nil {
						return err
					}
				}
				return nil
			})
		}

		assert.Check(t, group.Wait())
	})

	t.Run("the session still answers", func(t *testing.T) {
		_, err := client.Hover(doc, protocol.Position{Line: 3, Character: 8})
		assert.Check(t, err)
	})
}

// queryEverything makes each query the server answers once.
func queryEverything(client *lspclient.Client, doc uri.URI) error {
	document := protocol.TextDocumentIdentifier{URI: doc}
	at := protocol.TextDocumentPositionParams{
		TextDocument: document,
		Position:     protocol.Position{Line: 15, Character: 20},
	}

	queries := map[string]any{
		protocol.MethodTextDocumentHover:              protocol.HoverParams{TextDocumentPositionParams: at},
		protocol.MethodTextDocumentCompletion:         protocol.CompletionParams{TextDocumentPositionParams: at},
		protocol.MethodTextDocumentDefinition:         protocol.DefinitionParams{TextDocumentPositionParams: at},
		protocol.MethodTextDocumentReferences:         protocol.ReferenceParams{TextDocumentPositionParams: at},
		protocol.MethodTextDocumentSemanticTokensFull: protocol.SemanticTokensParams{TextDocument: document},
		protocol.MethodTextDocumentDocumentSymbol:     protocol.DocumentSymbolParams{TextDocument: document},
		protocol.MethodTextDocumentCodeAction: protocol.CodeActionParams{
			TextDocument: document,
			Context:      protocol.CodeActionContext{Diagnostics: []protocol.Diagnostic{}},
		},
	}
	for method, params := range queries {
		var result any
		if err := client.Call(method, params, &result); err != nil {
			return err
		}
	}
	return nil
}

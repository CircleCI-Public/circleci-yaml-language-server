package acceptance

import (
	"os"
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/workspace"
)

// symbolsConfig is the workspace's config.yml, as it is on disk.
const symbolsConfig = `version: 2.1

commands:
  greet:
    steps:
      - run: echo hello

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - greet
  deploy:
    docker:
      - image: cimg/base:current
    steps:
      - run: echo deploying

workflows:
  main:
    jobs:
      - build
      - deploy
`

// continueConfig is a config a setup workflow continues with, which is never
// opened.
const continueConfig = `version: 2.1

jobs:
  deploy-docs:
    docker:
      - image: cimg/base:current
    steps:
      - run: echo docs

workflows:
  docs:
    jobs:
      - deploy-docs
`

// symbol is what a test compares of a workspace symbol: its name, the
// section it is in, and the file it is declared in.
type symbol struct {
	Name, Container, File string
}

func symbolsOf(found []protocol.SymbolInformation) []symbol {
	symbols := []symbol{}
	for _, s := range found {
		container := ""
		if s.ContainerName != nil {
			container = *s.ContainerName
		}
		symbols = append(symbols, symbol{s.Name, container, filepath.Base(s.Location.URI.FsPath())})
	}
	return symbols
}

func TestWorkspaceSymbols(t *testing.T) {
	project := workspace.New(t, symbolsConfig)
	continuePath := filepath.Join(project.Root, ".circleci", "continue", "deploy.yml")
	err := os.MkdirAll(filepath.Dir(continuePath), 0o755)
	assert.NilError(t, err)
	err = os.WriteFile(continuePath, []byte(continueConfig), 0o600)
	assert.NilError(t, err)
	// YAML outside .circleci is some other tool's, even with a job in it.
	err = os.WriteFile(filepath.Join(project.Root, "deploy.yml"), []byte(continueConfig), 0o600)
	assert.NilError(t, err)

	session := startIn(t, linkedProjectFake(t), project, "")

	query := func(t *testing.T, text string) []symbol {
		t.Helper()
		found, err := session.client.WorkspaceSymbols(text)
		assert.NilError(t, err)
		return symbolsOf(found)
	}

	t.Run("finds the elements of every config, open or not", func(t *testing.T) {
		got := query(t, "")
		assert.Check(t, cmp.DeepEqual(got, []symbol{
			{"greet", "Commands", "config.yml"},
			{"build", "Jobs", "config.yml"},
			{"deploy", "Jobs", "config.yml"},
			{"main", "Workflows", "config.yml"},
			{"deploy-docs", "Jobs", "deploy.yml"},
			{"docs", "Workflows", "deploy.yml"},
		}))
	})

	t.Run("matches part of a name, in any case", func(t *testing.T) {
		got := query(t, "DEPLOY")
		assert.Check(t, cmp.DeepEqual(got, []symbol{
			{"deploy", "Jobs", "config.yml"},
			{"deploy-docs", "Jobs", "deploy.yml"},
		}))
	})

	t.Run("reads an open config as it is in the editor", func(t *testing.T) {
		edited := symbolsConfig + `
executors:
  linux:
    docker:
      - image: cimg/base:current
`
		session.open(t, edited)

		got := query(t, "linux")
		assert.Check(t, cmp.DeepEqual(got, []symbol{{"linux", "Executors", "config.yml"}}))
	})
}

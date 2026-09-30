package languageservice

import (
	"path/filepath"
	"testing"

	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestWorkspaceSymbolsLeavesOutAnUnreadableDocument(t *testing.T) {
	missing := uri.File(filepath.Join(t.TempDir(), ".circleci", "config.yml"))

	symbols := WorkspaceSymbols("", []uri.URI{missing}, testHelpers.DefaultCache(), testHelpers.DefaultSettings())
	assert.Check(t, cmp.Len(symbols, 0))
}

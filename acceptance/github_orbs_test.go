package acceptance

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/workspace"
)

// privateGitHubOrb is the URL of an orb in a private repository on GitHub,
// which GitHub won't serve without credentials.
const privateGitHubOrb = "https://raw.githubusercontent.com/acme/orbs/main/tools.yml"

// privateGitHubOrbConfig is urlOrbConfig with its orb in a private repository
// on GitHub.
var privateGitHubOrbConfig = strings.Replace(urlOrbConfig, "https://127.0.0.1:1/orbs/tools.yml", privateGitHubOrb, 1)

const signInWarning = "This orb could not be fetched: it doesn't exist, or it is private, so nothing used from it is checked." +
	" Sign in to GitHub to fetch private orbs from it."

// TestClientFetchesGitHubFiles is a client that fetches files from GitHub for
// the server, as the user's GitHub account in the IDE, rather than giving it
// a GitHub token.
func TestClientFetchesGitHubFiles(t *testing.T) {
	fake := linkedProjectFake(t)

	// gitHub answers 404 for everything, as GitHub does for a file in a
	// private repository asked for without credentials.
	gitHub := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(gitHub.Close)
	environment := []string{"LSP_GITHUB_URL=" + gitHub.URL, "GH_TOKEN=", "GITHUB_TOKEN="}

	fetchesGitHubFiles := map[string]any{
		"fetchesGitHubFiles":  true,
		"gitHubSignInCommand": "acme.signInToGitHub",
	}
	found := func(string) (any, error) {
		return map[string]any{"found": true, "source": acmeToolsOrbSource}, nil
	}
	notFound := func(string) (any, error) {
		return map[string]any{"found": false}, nil
	}

	t.Run("a private orb the client can see is checked", func(t *testing.T) {
		session := startWithOptions(t, fake, workspace.New(t, privateGitHubOrbConfig), testToken, fetchesGitHubFiles, environment...)
		session.client.AnswerGitHubFiles(found)

		diagnostics := session.open(t, privateGitHubOrbConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
		assert.Check(t, cmp.DeepEqual(session.client.GitHubFilesAsked(), []string{privateGitHubOrb}))
	})

	t.Run("an orb the client can't see offers to sign in", func(t *testing.T) {
		session := startWithOptions(t, fake, workspace.New(t, privateGitHubOrbConfig), testToken, fetchesGitHubFiles, environment...)
		session.client.AnswerGitHubFiles(notFound)

		var diagnostics []string
		t.Run("open the config", func(t *testing.T) {
			diagnostics = session.open(t, privateGitHubOrbConfig)
			assert.Check(t, cmp.DeepEqual(diagnostics, []string{signInWarning}))
		})

		t.Run("sign in, and the orb is asked for again", func(t *testing.T) {
			session.client.AnswerGitHubFiles(found)
			err := session.client.ExecuteCommand("gitHubAccountChanged")
			assert.NilError(t, err)

			diagnostics, err := session.client.WaitForDiagnostics(session.workspace.URI())
			assert.NilError(t, err)
			assert.Check(t, cmp.DeepEqual(messages(diagnostics), []string{}))
		})
	})

	t.Run("a client that doesn't fetch GitHub files is never asked", func(t *testing.T) {
		session := startWithOptions(t, fake, workspace.New(t, privateGitHubOrbConfig), testToken,
			map[string]any{"gitHubSignInCommand": "acme.signInToGitHub"}, environment...)
		session.client.AnswerGitHubFiles(found)

		diagnostics := session.open(t, privateGitHubOrbConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{signInWarning}))
		assert.Check(t, cmp.Len(session.client.GitHubFilesAsked(), 0))
	})
}

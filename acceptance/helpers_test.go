package acceptance

import (
	"testing"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"go.lsp.dev/protocol"
)

// settleTimeout bounds waiting for work the server does after it has answered:
// a didOpen publishes diagnostics and then carries on reading.
const settleTimeout = 20 * time.Second

// messages is what the diagnostics say, which is what these tests assert on.
// Ranges are the parser's business and are covered where the parser is tested,
// and a failure prints something readable.
func messages(diagnostics []protocol.Diagnostic) []string {
	said := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		said = append(said, diagnostic.MessageText(d))
	}

	return said
}

// eventually waits for something the server reaches on its own time, rather
// than in reply to a request.
func eventually(t *testing.T, what string, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(settleTimeout)

	for {
		if condition() {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("timed out after %s waiting for %s", settleTimeout, what)
		}

		time.Sleep(20 * time.Millisecond)
	}
}

func position(line, character uint32) protocol.Position {
	return protocol.Position{Line: line, Character: character}
}

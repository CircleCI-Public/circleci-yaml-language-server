package parser_test

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
)

func TestMachineTrueMessages(t *testing.T) {
	messages := map[string]string{
		"linux":   parser.MachineTrueMessage("ubuntu-2404:current"),
		"windows": parser.MachineTrueWindowsMessage,
	}

	for name, message := range messages {
		t.Run(name, func(t *testing.T) {
			assert.Check(t, !strings.Contains(message, "\t"), "tabs aren't valid YAML indentation")

			for i, line := range strings.Split(message, "\n") {
				assert.Check(t, cmp.Equal(line, strings.TrimRight(line, " ")), "line %d has trailing spaces", i)
			}
		})
	}

	t.Run("the suggested declaration is valid YAML", func(t *testing.T) {
		message := parser.MachineTrueMessage("ubuntu-2404:current")
		assert.Check(t, cmp.Contains(message, "\nmachine:\n  image: ubuntu-2404:current"))
	})
}

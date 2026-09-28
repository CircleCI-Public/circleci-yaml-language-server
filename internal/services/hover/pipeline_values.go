package hover

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/pipelinevalues"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

var pipelineValueName = regexp.MustCompile(`\bpipeline(?:\.[A-Za-z0-9_]+)+`)

// PipelineValue is the hover for a built-in pipeline value, such as
// pipeline.git.branch: its type and what it holds.
func PipelineValue(doc yamlparser.YamlDocument, _ *cache.Cache, pos protocol.Position) (string, bool) {
	lineStart := position.ToIndex(protocol.Position{Line: pos.Line}, doc.Content)
	lineEnd := bytes.IndexByte(doc.Content[lineStart:], '\n')
	if lineEnd == -1 {
		lineEnd = len(doc.Content) - lineStart
	}
	line := doc.Content[lineStart : lineStart+lineEnd]
	column := int(pos.Character)

	for _, match := range pipelineValueName.FindAllIndex(line, -1) {
		if column < match[0] || column > match[1] {
			continue
		}
		value, ok := pipelinevalues.Lookup(string(line[match[0]:match[1]]))
		if !ok {
			return "", false
		}

		var b strings.Builder
		fmt.Fprintf(&b, "**%s** `%s`", value.Name, value.Type)
		if value.Definition != "" {
			fmt.Fprintf(&b, "\n\n%s", value.Definition)
		}
		if value.ReplacedBy != "" {
			fmt.Fprintf(&b, "\n\nDeprecated: use `%s` instead.", value.ReplacedBy)
		}
		return b.String(), true
	}
	return "", false
}

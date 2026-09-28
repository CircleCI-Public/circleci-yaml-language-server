package validate

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/template"
)

// ValidateTemplates reports the `<< >>` tags and expressions the compiler
// can't parse, such as an unescaped heredoc, `cat <<EOF`.
func (val Validate) ValidateTemplates() {
	if val.Doc.Version < 2.1 {
		return
	}

	check := func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := &capture.Node
			// `<<` alone is a YAML merge key.
			if val.Doc.IsUnderUnreadTopLevelKey(node) || val.Doc.GetRawNodeText(node) == "<<" {
				continue
			}
			problem, found := template.Check(val.Doc.GetRawNodeText(node))
			if !found {
				continue
			}
			start := int(node.StartByte())
			val.addDiagnostic(diagnostic.Error(protocol.Range{
				Start: position.FromIndex(start+problem.Start, val.Doc.Content),
				End:   position.FromIndex(start+problem.End, val.Doc.Content),
			}, problem.Message))
		}
	}

	stringScalarsQuery.Run(val.Doc.RootNode, check)
	blockScalarsQuery.Run(val.Doc.RootNode, check)
	quotedScalarsQuery.Run(val.Doc.RootNode, check)
}

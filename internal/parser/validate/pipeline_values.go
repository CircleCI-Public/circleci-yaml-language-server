package validate

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/pipelinevalues"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/template"
)

// ValidatePipelineValues warns about a pipeline value that doesn't exist,
// which the compiler leaves empty, and about one that's been replaced. They
// can be in a `<< >>` tag, or in a condition or filter written as an
// expression.
func (val Validate) ValidatePipelineValues() {
	if val.Doc.Version < 2.1 {
		return
	}

	inTags := func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := &capture.Node
			if val.Doc.IsUnderUnreadTopLevelKey(node) {
				continue
			}
			start := int(node.StartByte())
			for _, reference := range template.References(val.Doc.GetRawNodeText(node)) {
				val.checkPipelineValue(reference.Name, protocol.Range{
					Start: position.FromIndex(start+reference.Start, val.Doc.Content),
					End:   position.FromIndex(start+reference.End, val.Doc.Content),
				})
			}
		}
	}
	stringScalarsQuery.Run(val.Doc.RootNode, inTags)
	blockScalarsQuery.Run(val.Doc.RootNode, inTags)
	quotedScalarsQuery.Run(val.Doc.RootNode, inTags)

	inExpressions := func(values []ast.TextAndRange) {
		for _, value := range values {
			if !val.isExpression(value) {
				continue
			}
			references, _ := template.Expression(value.Text)
			for _, reference := range references {
				val.checkPipelineValue(reference.Name, val.rangeWithin(value.Range, reference.Start, reference.End))
			}
		}
	}
	inExpressions(val.Doc.Conditions)
	inExpressions(val.Doc.FilterExpressions)
}

func (val Validate) checkPipelineValue(name string, rng protocol.Range) {
	if !strings.HasPrefix(name, "pipeline.") || strings.HasPrefix(name, "pipeline.parameters.") {
		return
	}

	value, known := pipelinevalues.Lookup(name)
	switch {
	case !known:
		val.addDiagnostic(diagnostic.Warning(rng, fmt.Sprintf("Unknown pipeline value %s", name)))
	case value.ReplacedBy != "":
		val.addDiagnostic(diagnostic.Deprecated(rng, fmt.Sprintf("%s is deprecated, use %s instead", name, value.ReplacedBy)))
	}
}

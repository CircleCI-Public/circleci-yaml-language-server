package validate

import (
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamlbool"
)

func (val Validate) createParameterError(param ast.ParameterValue, stepName string, shouldBeType string) {
	message := fmt.Sprintf("Parameter %s for %s must be a %s", param.Name, stepName, shouldBeType)
	if text := val.textAt(param.ValueRange); param.Type == "boolean" && yamlbool.IsYAML11(text) {
		message += fmt.Sprintf(". `%s` is read as a boolean; quote it if it's meant as text.", text)
	}
	*val.Diagnostics = append(*val.Diagnostics, diagnostic.Error(param.Range, message))
}

func (val Validate) textAt(rng protocol.Range) string {
	start := position.ToIndex(rng.Start, val.Doc.Content)
	end := position.ToIndex(rng.End, val.Doc.Content)
	if start < 0 || end > len(val.Doc.Content) || start > end {
		return ""
	}
	return string(val.Doc.Content[start:end])
}

func (val Validate) addDiagnostic(diag protocol.Diagnostic) {
	*val.Diagnostics = append(*val.Diagnostics, diag)
}

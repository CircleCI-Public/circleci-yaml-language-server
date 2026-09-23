package validate

import (
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

func (val Validate) createParameterError(param ast.ParameterValue, stepName string, shouldBeType string) {
	*val.Diagnostics = append(*val.Diagnostics, diagnostic.Error(
		param.Range,
		fmt.Sprintf("Parameter %s for %s must be a %s", param.Name, stepName, shouldBeType)),
	)
}

func (val Validate) addDiagnostic(diag protocol.Diagnostic) {
	*val.Diagnostics = append(*val.Diagnostics, diag)
}

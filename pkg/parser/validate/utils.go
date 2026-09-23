package validate

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"go.lsp.dev/protocol"
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

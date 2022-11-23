package validate

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (val Validate) createParameterError(param ast.ParameterValue, stepName string, shouldBeType string) {
	*val.Diagnostics = append(*val.Diagnostics, utils.CreateErrorDiagnosticFromRange(
		param.Range,
		fmt.Sprintf("Parameter %s for %s must be a %s", param.Name, stepName, shouldBeType)),
	)
}

func (val Validate) addDiagnostic(diagnostic protocol.Diagnostic) {
	*val.Diagnostics = append(*val.Diagnostics, diagnostic)
}

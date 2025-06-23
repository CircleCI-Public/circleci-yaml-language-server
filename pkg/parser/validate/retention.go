package validate

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
)

func (val Validate) validateRetention(retention ast.RetentionSettings) {
	diagnostics := retention.ValidateCaches()
	for _, diag := range diagnostics {
		val.addDiagnostic(diag)
	}
}

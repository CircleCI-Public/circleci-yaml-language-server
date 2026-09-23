package validate

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

func (val Validate) checkEnumTypeDefinition(definedParam ast.EnumParameter) {
	if definedParam.HasDefault {
		if !slices.Contains(definedParam.Enum, definedParam.Default) {
			val.addDiagnostic(diagnostic.Error(
				definedParam.Range,
				fmt.Sprintf("Default value %s is not in enum '%s'", definedParam.Default, strings.Join(definedParam.Enum, ", "))))
		}
	}
}

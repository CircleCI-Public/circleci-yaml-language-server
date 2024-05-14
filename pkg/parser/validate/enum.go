package validate

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func (val Validate) checkEnumTypeDefinition(definedParam ast.EnumParameter) {
	if definedParam.HasDefault {
		if !slices.Contains(definedParam.Enum, definedParam.Default) {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
				definedParam.Range,
				fmt.Sprintf("Default value %s is not in enum '%s'", definedParam.Default, strings.Join(definedParam.Enum, ", "))))
		}
	}
}

package validate

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func (val Validate) checkEnumTypeDefinition(definedParam ast.EnumParameter) {
	if definedParam.HasDefault {
		if utils.FindInArray(definedParam.Enum, definedParam.Default) == -1 {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
				definedParam.Range,
				fmt.Sprintf("Default value %s is not in enum %s", definedParam.Default, definedParam.Name)))
		}
	}
}

package validate

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"

	"go.lsp.dev/protocol"
)

type Validate struct {
	Diagnostics *[]protocol.Diagnostic
	Doc         parser.YamlDocument
	Cache       *utils.Cache
	Context     *utils.LsContext
}

func (val *Validate) Validate() {
	val.ValidateAnchors()
	val.CheckIfParamsExist()
	val.ValidateWorkflows()
	val.ValidateJobs()
	val.ValidateCommands()
	val.ValidateOrbs()
	val.ValidateExecutors()
	val.CheckNames()
}

package validate

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"

	"go.lsp.dev/protocol"
)

type ValidateAPIs struct {
	DockerHub dockerhub.DockerHubAPI
}

type Validate struct {
	APIs        ValidateAPIs
	Diagnostics *[]protocol.Diagnostic
	Doc         parser.YamlDocument
	Cache       *utils.Cache
	Context     *utils.LsContext
	IsLocalOrb  bool
}

func (val *Validate) Validate() {
	if !val.IsLocalOrb {
		val.CheckIfParamsExist()
		val.ValidateAnchors()
		val.ValidateWorkflows()
		val.ValidateOrbs()
		val.CheckNames()
		val.ValidatePipelineParameters()
		val.ValidateLocalOrbs()
	}
	val.ValidateJobs()
	val.ValidateCommands()
	val.ValidateExecutors()
}

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
}

func (val *Validate) Validate(inLocalOrb bool) {
	val.ValidateAnchors()
	if !inLocalOrb {
		val.CheckIfParamsExist()
	}
	val.ValidateWorkflows()
	val.ValidateJobs()
	val.ValidateCommands()
	val.ValidateOrbs()
	val.ValidateExecutors()
	val.CheckNames()
	val.ValidatePipelineParameters()
	val.ValidateLocalOrbs()
}

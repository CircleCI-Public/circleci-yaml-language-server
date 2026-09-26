package validate

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

type ValidateAPIs struct {
	DockerHub dockerhub.API
}

type Validate struct {
	APIs        ValidateAPIs
	Diagnostics *[]protocol.Diagnostic
	Doc         parser.YamlDocument
	Cache       *cache.Cache
	Context     *session.Settings
	IsLocalOrb  bool
}

func (val *Validate) Validate() {
	if !val.IsLocalOrb {
		val.CheckIfParamsExist()
		val.ValidateAnchors()
		val.ValidateJobGroups()
		val.ValidateWorkflows()
		val.ValidateOrbs()
		val.CheckNames()
		val.ValidatePipelineParameters()
		val.ValidateLocalOrbs()
		val.ValidateFunctions()
		val.ValidateConditions()
		val.ValidateRegexes()
	}
	val.ValidateJobs()
	val.ValidateCommands()
	val.ValidateExecutors()
}

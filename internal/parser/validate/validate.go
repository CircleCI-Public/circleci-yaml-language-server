package validate

import (
	"context"

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
	// Outer is the config an inline orb is declared in, as OrbName.
	Outer   *parser.YamlDocument
	OrbName string
}

func (val *Validate) Validate(ctx context.Context) {
	if !val.IsLocalOrb {
		val.CheckIfParamsExist(ctx)
		val.ValidateAnchors()
		val.ValidateJobGroups(ctx)
		val.ValidateWorkflows(ctx)
		val.ValidateOrbs(ctx)
		val.CheckNames()
		val.ValidatePipelineParameters()
		val.ValidateLocalOrbs(ctx)
		val.ValidateFunctions(ctx)
		val.ValidateConditions()
		val.ValidateRegexes()
		val.ValidateMatchesValues()
		val.ValidateTemplates()
		val.ValidatePipelineValues()
	}
	val.ValidateJobs(ctx)
	val.ValidateCommands(ctx)
	val.ValidateExecutors(ctx)
	val.ValidateTypedKeyReferences()
}

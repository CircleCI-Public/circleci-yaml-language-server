package complete

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// completeInInlineOrb completes in the body of an inline orb with the same
// completers as the config's own sections, and reports whether the cursor is
// in one.
func (ch *CompletionHandler) completeInInlineOrb() bool {
	for _, orb := range ch.Doc.Orbs {
		if !orb.Url.IsLocal || !position.InRange(orb.ValueRange, ch.Params.Position) {
			continue
		}

		if info, ok := ch.Doc.LocalOrbInfo[orb.Name]; ok {
			config := ch.Doc
			ch.Doc = orbScope(config, info)
			ch.completeSection()
			ch.Doc = config
		}
		return true
	}
	return false
}

// orbScope is the config narrowed to what an inline orb can refer to: its
// own jobs, commands and executors, which it names without the orb's prefix.
// The config's orbs aren't in scope, and neither are its workflows, job groups
// and functions, which an orb can't have.
func orbScope(config yamlparser.YamlDocument, orb *ast.OrbInfo) yamlparser.YamlDocument {
	scoped := config

	scoped.Orbs = nil
	scoped.LocalOrbInfo = nil
	scoped.Functions = nil
	scoped.JobGroups = nil
	scoped.Workflows = nil

	scoped.Jobs = orb.Jobs
	scoped.Commands = orb.Commands
	scoped.Executors = orb.Executors
	scoped.PipelineParameters = orb.PipelineParameters

	scoped.JobsRange = orb.JobsRange
	scoped.CommandsRange = orb.CommandsRange
	scoped.ExecutorsRange = orb.ExecutorsRange
	scoped.PipelineParametersRange = orb.PipelineParametersRange
	scoped.OrbsRange = protocol.Range{}
	scoped.WorkflowRange = protocol.Range{}
	scoped.JobGroupsRange = protocol.Range{}
	scoped.FunctionsRange = protocol.Range{}

	return scoped
}

package validate

import (
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

type namedEntity struct {
	name      string
	nameRange protocol.Range
	kind      string
}

// Adds diagnostics for duplicate names of workflows, jobs,
// commands, and job-groups in the config.
func (val Validate) CheckNames() {
	var entities []namedEntity

	for _, w := range val.Doc.Workflows {
		entities = append(entities, namedEntity{w.Name, w.NameRange, "workflow"})
	}
	for _, j := range val.Doc.Jobs {
		entities = append(entities, namedEntity{j.Name, j.NameRange, "job"})
	}
	for _, c := range val.Doc.Commands {
		entities = append(entities, namedEntity{c.Name, c.NameRange, "command"})
	}
	for _, jg := range val.Doc.JobGroups {
		entities = append(entities, namedEntity{jg.Name, jg.NameRange, "job-group"})
	}

	for i := 0; i < len(entities); i++ {
		for j := i + 1; j < len(entities); j++ {
			a, b := entities[i], entities[j]
			if a.name == b.name && a.kind != b.kind {
				val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(
					a.nameRange,
					fmt.Sprintf("The name \"%s\" is already used to define a %s. You might want to use a different name to avoid confusion.", a.name, b.kind)))
				val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(
					b.nameRange,
					fmt.Sprintf("The name \"%s\" is already used to define a %s. You might want to use a different name to avoid confusion.", b.name, a.kind)))
			}
		}
	}
}

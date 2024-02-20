/**
 * Before explaining this code, it is important to know that compared to a normal yaml, an orb can
 * only declare three type of things:
 *  - commands
 *  - executors
 *  - jobs
 *
 *
 * The parsing of local orbs is as is:
 *  - convert the orb's content into a string
 *  - remove the orb's indentation so that is has the format of a real yaml
 *  - parse the orb's yaml with GetParsedYAMLWithContent
 *  - go through all the entities that are declared inside the orb and move their range according
 *    to the orb's position
 *    We do this because when GetParsedYAMLWithContent parsed the orb's yaml it does so by starting
 *    at line 0 and character 0, so all the entities need to be offseted well to match their real
 *    position in the global yaml config file
 *  - prefix the name of all the commands, executors and jobs with `{orbName}/` and add them to the
 *    global yaml scope
 *
 * Now the entities declared in the orb can be accessed with `{orbName}/{entityName}`
 *
 * This solution is possible because you can not create any commands, executor or jobs that starts
 * with `prefix/`
 * To convince yourself of this:
 *  - for commands: https://app.circleci.com/pipelines/github/circleci/circleci-vscode-extension/422
 *  - for executors: https://app.circleci.com/pipelines/github/circleci/circleci-vscode-extension/420
 *  - for jobs: we don't allow '/' in job names
 */

package parser

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

type LocalOrb struct {
	Name string
}

func (doc *YamlDocument) parseLocalOrb(name string, orbNode *sitter.Node) (*LocalOrb, error) {
	orbRange := doc.NodeToRange(orbNode)
	orb := LocalOrb{
		Name: name,
	}

	if orbNode.Type() != "block_node" {
		return nil, fmt.Errorf("Invalid orb body")
	}

	orbContent := strings.Repeat(" ", int(orbRange.Start.Character)) + doc.GetNodeText(orbNode)
	orbDoc, err := ParseFromContent([]byte(orbContent), doc.Context, doc.URI, protocol.Position{
		Line:      orbRange.Start.Line,
		Character: 0,
	})

	if err != nil {
		return nil, err
	}

	orbInfo := &ast.OrbInfo{
		IsLocal: true,

		Source:      orbContent,
		Description: orbDoc.Description,
		OrbParsedAttributes: ast.OrbParsedAttributes{
			URI:  doc.URI,
			Name: name,

			Commands:           orbDoc.Commands,
			Jobs:               orbDoc.Jobs,
			Executors:          orbDoc.Executors,
			PipelineParameters: orbDoc.PipelineParameters,

			OrbsRange:               orbDoc.OrbsRange,
			ExecutorsRange:          orbDoc.ExecutorsRange,
			CommandsRange:           orbDoc.CommandsRange,
			JobsRange:               orbDoc.JobsRange,
			WorkflowRange:           orbDoc.WorkflowRange,
			PipelineParametersRange: orbDoc.PipelineParametersRange,
		},
	}

	doc.LocalOrbInfo[name] = orbInfo

	// Diagnostics
	for _, diagnostic := range *orbDoc.Diagnostics {
		doc.addDiagnostic(diagnostic)
	}

	return &orb, nil
}

func (doc *YamlDocument) getAttributeName(attribute string) string {
	if doc.LocalOrbName != "" {
		return fmt.Sprintf("%s/%s", doc.LocalOrbName, attribute)
	}
	return attribute
}

package definition

import (
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
)

func (def DefinitionStruct) getCommandOrJobLocation(name string, includeCommands bool) ([]protocol.Location, error) {
	// The order of these checks is important. If a job, job-group,
	// and command all have the same name, this function will return
	// the job first (of course, there will also be a warning diagnostic
	// about the ambiguous names).
	if job, ok := def.Doc.Jobs[name]; ok {
		return []protocol.Location{
			{
				Range: job.Range,
				URI:   def.Doc.URI,
			},
		}, nil
	}

	if jobGroup, ok := def.Doc.JobGroups[name]; ok {
		return []protocol.Location{
			{
				Range: jobGroup.Range,
				URI:   def.Doc.URI,
			},
		}, nil
	}

	if command, ok := def.Doc.Commands[name]; ok && includeCommands {
		return []protocol.Location{
			{
				Range: command.Range,
				URI:   def.Doc.URI,
			},
		}, nil
	}

	if orb, err := def.getOrbLocation(name, true); err == nil {
		return orb, nil
	}

	return []protocol.Location{}, fmt.Errorf("command or job not found")
}

func (def DefinitionStruct) getCommandOrJobParamLocation(name string, paramName string, includeCommands bool) ([]protocol.Location, error) {
	if job, ok := def.Doc.Jobs[name]; ok {
		if param, ok := job.Parameters[paramName]; ok {
			return []protocol.Location{
				{
					Range: param.GetRange(),
					URI:   def.Doc.URI,
				},
			}, nil
		}
	}

	if command, ok := def.Doc.Commands[name]; ok && includeCommands {
		if param, ok := command.Parameters[paramName]; ok {
			return []protocol.Location{
				{
					Range: param.GetRange(),
					URI:   def.Doc.URI,
				},
			}, nil
		}
	}

	if orb, err := def.getOrbParamLocation(name, paramName); err == nil {
		return orb, nil
	}

	return []protocol.Location{}, fmt.Errorf("command or job not found")
}

func GetPathFromVisitedNodes(visitedNodes []*sitter.Node, doc parser.YamlDocument) []string {
	var path []string
	if len(visitedNodes) == 0 {
		return path
	}

	for _, node := range visitedNodes[1:] {
		switch node.Kind() {
		case "block_mapping_pair":
			// A pair whose key is yet to be typed, as `: value` is, has no
			// name to go in the path.
			key := node.ChildByFieldName("key")
			if key == nil {
				continue
			}
			name := string(doc.Content[key.StartByte():key.EndByte()])
			path = append(path, name)
		case "block_sequence_item":
			flow_node := parser.GetChildOfType(node, "flow_node")
			if flow_node != nil {
				name := string(doc.Content[flow_node.StartByte():flow_node.EndByte()])
				path = append(path, name)
			}
		}
	}

	return path
}

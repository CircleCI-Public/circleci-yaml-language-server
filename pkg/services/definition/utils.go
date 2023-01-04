package definition

import (
	"fmt"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (def DefinitionStruct) getCommandOrJobLocation(name string, includeCommands bool) ([]protocol.Location, error) {
	if job, ok := def.Doc.Jobs[name]; ok {
		return []protocol.Location{
			{
				Range: job.Range,
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

func GetPathFromVisitedNodes(visitedNodes []*sitter.Node, doc yamlparser.YamlDocument) []string {
	var path []string
	if len(visitedNodes) == 0 {
		return path
	}

	for _, node := range visitedNodes[1:] {
		switch node.Type() {
		case "block_mapping_pair":
			key := node.ChildByFieldName("key")
			name := string(doc.Content[key.StartByte():key.EndByte()])
			path = append(path, name)
		case "block_sequence_item":
			flow_node := yamlparser.GetChildOfType(node, "flow_node")
			if flow_node != nil {
				name := string(doc.Content[flow_node.StartByte():flow_node.EndByte()])
				path = append(path, name)
			}
		}
	}

	return path
}

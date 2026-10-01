package definition

import (
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
)

func (def DefinitionStruct) getCommandOrJobLocation(name string, includeCommands bool) ([]Link, error) {
	// The order of these checks is important. If a job, job-group,
	// and command all have the same name, this function will return
	// the job first (of course, there will also be a warning diagnostic
	// about the ambiguous names).
	if job, ok := def.Doc.Jobs[name]; ok {
		return []Link{{URI: def.Doc.URI, Range: job.Range, NameRange: job.NameRange}}, nil
	}

	if jobGroup, ok := def.Doc.JobGroups[name]; ok {
		return []Link{{URI: def.Doc.URI, Range: jobGroup.Range, NameRange: jobGroup.NameRange}}, nil
	}

	if command, ok := def.Doc.Commands[name]; ok && includeCommands {
		return []Link{{URI: def.Doc.URI, Range: command.Range, NameRange: command.NameRange}}, nil
	}

	if alias, ok := def.Doc.JobAlias(name); ok {
		return []Link{{URI: def.Doc.URI, Range: alias.Range, NameRange: alias.NameRange}}, nil
	}
	if alias, ok := def.Doc.CommandAlias(name); ok && includeCommands {
		return []Link{{URI: def.Doc.URI, Range: alias.Range, NameRange: alias.NameRange}}, nil
	}

	if orb, err := def.getOrbLocation(name, true); err == nil {
		return orb, nil
	}

	return []Link{}, fmt.Errorf("command or job not found")
}

func (def DefinitionStruct) getCommandOrJobParamLocation(name string, paramName string, includeCommands bool) ([]Link, error) {
	// An alias's arguments are its target's parameters.
	if alias, ok := def.Doc.CommandAlias(name); ok && includeCommands {
		name = alias.Target
	} else if alias, ok := def.Doc.JobAlias(name); ok {
		name = alias.Target
	}

	if job, ok := def.Doc.Jobs[name]; ok {
		if param, ok := job.Parameters[paramName]; ok {
			return []Link{{URI: def.Doc.URI, Range: param.GetRange(), NameRange: param.GetNameRange()}}, nil
		}
	}

	if command, ok := def.Doc.Commands[name]; ok && includeCommands {
		if param, ok := command.Parameters[paramName]; ok {
			return []Link{{URI: def.Doc.URI, Range: param.GetRange(), NameRange: param.GetNameRange()}}, nil
		}
	}

	if orb, err := def.getOrbParamLocation(name, paramName); err == nil {
		return orb, nil
	}

	return []Link{}, fmt.Errorf("command or job not found")
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

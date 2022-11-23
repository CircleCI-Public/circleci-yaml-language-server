package parser

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (doc *YamlDocument) parseCommands(commandsNode *sitter.Node) {
	// commandsNode is of type block_node
	blockMappingNode := GetChildMapping(commandsNode)
	if blockMappingNode == nil {
		return
	}

	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		command := doc.parseSingleCommand(child)
		if definedCommand, ok := doc.Commands[command.Name]; ok {
			doc.addDiagnostic(protocol.Diagnostic{
				Severity: protocol.DiagnosticSeverityWarning,
				Range:    command.NameRange,
				Message:  "Command already defined",
				Source:   "cci-language-server",
			})
			doc.addDiagnostic(protocol.Diagnostic{
				Severity: protocol.DiagnosticSeverityWarning,
				Range:    definedCommand.NameRange,
				Message:  "Command already defined",
				Source:   "cci-language-server",
			})

			return
		}

		if command.Name != "" {
			doc.Commands[command.Name] = command
		}
	})
}

func (doc *YamlDocument) parseSingleCommand(commandNode *sitter.Node) ast.Command {
	// commandNode is a block_mapping_pair
	commandNameNode, blockMappingNode := getKeyValueNodes(commandNode)
	res := ast.Command{}
	if commandNameNode == nil || blockMappingNode == nil {
		return res
	}
	commandName := doc.GetNodeText(commandNameNode)
	blockMappingNode = GetChildMapping(blockMappingNode)

	if blockMappingNode == nil { //TODO: deal with errors
		return res
	}
	res.Name = commandName
	res.Range = NodeToRange(commandNode)
	res.NameRange = NodeToRange(commandNameNode)

	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		if valueNode == nil {
			return
		}
		switch keyName {
		case "description":
			res.DescriptionRange = NodeToRange(child)
			res.Description = doc.parseDescription(valueNode)
		case "steps":
			res.StepsRange = NodeToRange(valueNode)
			res.Steps = doc.parseSteps(valueNode)
		case "parameters":
			res.ParametersRange = NodeToRange(valueNode)
			res.Parameters = doc.parseParameters(valueNode)
		}
	})

	return res
}

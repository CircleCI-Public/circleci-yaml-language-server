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

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
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
	commandNameNode, blockMappingNode := doc.GetKeyValueNodes(commandNode)
	res := ast.Command{Contexts: &[]string{}, Parameters: map[string]ast.Parameter{}}
	if commandNameNode == nil || blockMappingNode == nil {
		return res
	}
	commandName := doc.GetNodeText(commandNameNode)
	blockMappingNode = GetChildMapping(blockMappingNode)

	if blockMappingNode == nil { //TODO: deal with errors
		return res
	}
	res.Name = doc.getAttributeName(commandName)
	res.Range = doc.NodeToRange(commandNode)
	res.NameRange = doc.NodeToRange(commandNameNode)

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)
		if valueNode == nil {
			return
		}
		switch keyName {
		case "description":
			res.DescriptionRange = doc.NodeToRange(child)
			res.Description = doc.parseDescription(valueNode)
		case "steps":
			res.StepsRange = doc.NodeToRange(valueNode)
			res.Steps = doc.parseSteps(valueNode)
		case "parameters":
			res.ParametersRange = doc.NodeToRange(valueNode)
			res.Parameters = doc.parseParameters(valueNode)
		}
	})

	return res
}

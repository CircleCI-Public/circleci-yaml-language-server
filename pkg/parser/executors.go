package parser

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (doc *YamlDocument) parseExecutors(executorsNode *sitter.Node) {
	// executorsNode is a block_node
	blockMappingNode := GetChildMapping(executorsNode)
	if blockMappingNode == nil {
		return
	}

	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		doc.parseSingleExecutor(child)
	})
}

func (doc *YamlDocument) parseSingleExecutor(executorNode *sitter.Node) {
	// jobNode is a block_mapping_pair
	executorNameNode := executorNode.ChildByFieldName("key")
	executorName := doc.GetNodeText(executorNameNode)
	blockMappingNode := executorNode.ChildByFieldName("value")
	blockMappingNode = GetChildMapping(blockMappingNode)
	if blockMappingNode == nil {
		return
	}

	if definedExecutor, ok := doc.Executors[executorName]; ok {
		doc.addDiagnostic(protocol.Diagnostic{
			Severity: protocol.DiagnosticSeverityWarning,
			Range:    NodeToRange(executorNameNode),
			Message:  "Executor already defined",
			Source:   "cci-language-server",
		})
		doc.addDiagnostic(protocol.Diagnostic{
			Severity: protocol.DiagnosticSeverityWarning,
			Range:    definedExecutor.GetNameRange(),
			Message:  "Executor already defined",
			Source:   "cci-language-server",
		})

		return
	}

	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))

		switch keyName {
		case "docker":
			doc.Executors[executorName] = doc.parseSingleExecutorDocker(executorNameNode, blockMappingNode)
		case "machine":
			doc.Executors[executorName] = doc.parseSingleExecutorMachine(executorNameNode, blockMappingNode)
		case "macos":
			doc.Executors[executorName] = doc.parseSingleExecutorMacOS(executorNameNode, blockMappingNode)
		case "windows":
			doc.Executors[executorName] = doc.parseSingleExecutorWindows(executorNameNode, blockMappingNode)
		}
	})

	// If the executor has not been parsed and set in doc.Executors,
	// we can assume it is not complete and therefor we set it as uncomplete
	// and parse what has been set for the autocomplete to suggest the missing fields
	if _, ok := doc.Executors[executorName]; !ok {
		baseExecutor := ast.BaseExecutor{
			UserParameters: make(map[string]ast.Parameter),
		}
		doc.parseBaseExecutor(&baseExecutor, executorNameNode, blockMappingNode, func(node *sitter.Node) {}, "dummy")
		baseExecutor.Uncomplete = true
		doc.Executors[executorName] = baseExecutor
	}
}

func (doc *YamlDocument) parseBaseExecutor(base *ast.BaseExecutor, nameNode *sitter.Node, blockMappingNode *sitter.Node, fn func(node *sitter.Node), nameStep string) {
	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {

		case nameStep:
			fn(valueNode)

		case "resource_class":
			base.ResourceClass = doc.GetNodeText(valueNode)
			base.ResourceClassRange = NodeToRange(child)
			if base.ResourceClass == "" {
				base.ResourceClassRange.End.Character = 999
			}
		case "shell":
			base.BuiltInParameters.Shell = doc.GetNodeText(valueNode)
		case "working_directory":
			base.BuiltInParameters.WorkingDirectory = doc.GetNodeText(valueNode)
		case "environment":
			base.BuiltInParameters.Environment = doc.parseDictionary(GetChildOfType(valueNode, "block_mapping"))
		case "parameters":
			base.UserParametersRange = NodeToRange(child)
			base.UserParameters = doc.parseParameters(valueNode)
		}
	})

	base.Name = doc.GetNodeText(nameNode)
	base.NameRange = NodeToRange(nameNode)
	// We get the range of the parent of the parent to get the
	// whole definition of the executor (name and definition) and not only
	// the definition
	if blockMappingNode == nil || blockMappingNode.Parent() == nil || blockMappingNode.Parent().Parent() == nil {
		base.Uncomplete = true
		return
	}
	base.Range = NodeToRange(blockMappingNode.Parent().Parent())
	base.Uncomplete = false
}

func (doc *YamlDocument) parseSingleExecutorMachine(nameNode *sitter.Node, valueNode *sitter.Node) ast.MachineExecutor {
	// valueNode is a block_mapping
	res := ast.MachineExecutor{
		IsDeprecated: false,
	}

	parseMachine := func(blockNode *sitter.Node) {
		// This only happens when the executor is `machine: true`
		if doc.addedMachineTrueDeprecatedDiag(blockNode.Parent()) {
			res.IsDeprecated = true
			return
		}

		// blockNode is a block_node
		blockMappingNode := GetChildMapping(blockNode)

		if blockMappingNode == nil {
			return
		}

		iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			keyName := doc.GetNodeText(child.ChildByFieldName("key"))
			valueNode := child.ChildByFieldName("value")
			switch keyName {
			case "image":
				res.ImageRange = NodeToRange(child)
				res.Image = doc.GetNodeText(valueNode)
			case "docker_layer_caching":
				res.DockerLayerCaching = doc.GetNodeText(valueNode) == "true"
			case "resource_class":
				res.ResourceClassRange = NodeToRange(child)
				res.ResourceClass = doc.GetNodeText(valueNode)
			}
		})
	}

	doc.parseBaseExecutor(&res.BaseExecutor, nameNode, valueNode, parseMachine, "machine")

	return res
}

func (doc *YamlDocument) parseSingleExecutorMacOS(nameNode *sitter.Node, valueNode *sitter.Node) ast.MacOSExecutor {
	// valueNode is a block_mapping
	res := ast.MacOSExecutor{}

	parseMacOS := func(blockNode *sitter.Node) {
		// blockNode is a block_node
		blockMappingNode := GetChildMapping(blockNode)

		iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			keyName := doc.GetNodeText(child.ChildByFieldName("key"))
			valueNode := child.ChildByFieldName("value")
			switch keyName {
			case "xcode":
				res.XcodeRange = NodeToRange(child)
				res.Xcode = doc.GetNodeText(valueNode)
			}
		})
	}

	doc.parseBaseExecutor(&res.BaseExecutor, nameNode, valueNode, parseMacOS, "macos")
	return res
}

func (doc *YamlDocument) parseSingleExecutorWindows(nameNode *sitter.Node, valueNode *sitter.Node) ast.WindowsExecutor {
	// valueNode is a block_mapping
	res := ast.WindowsExecutor{}

	parseWindows := func(blockNode *sitter.Node) {
		// blockNode is a block_node
		blockMappingNode := GetChildMapping(blockNode)

		if blockMappingNode == nil { //TODO: deal with errors
			return
		}

		iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			keyName := doc.GetNodeText(child.ChildByFieldName("key"))
			valueNode := child.ChildByFieldName("value")
			switch keyName {
			case "image":
				res.Image = doc.GetNodeText(valueNode)
			}
		})
	}

	doc.parseBaseExecutor(&res.BaseExecutor, nameNode, valueNode, parseWindows, "windows")
	return res
}

func (doc *YamlDocument) parseSingleExecutorDocker(nameNode *sitter.Node, valueNode *sitter.Node) ast.DockerExecutor {
	// valueNode is a block_mapping
	res := ast.DockerExecutor{
		Image: make([]ast.DockerImage, 0),
	}

	parseDocker := func(blockNode *sitter.Node) {
		// blockNode is a block_node
		blockSequence := GetChildOfType(blockNode, "block_sequence")

		if blockSequence == nil { //TODO: deal with errors
			return
		}

		iterateOnBlockSequence(blockSequence, func(child *sitter.Node) {
			if child.Type() == "block_sequence_item" {
				dockerImg := doc.parseDockerImage(child)
				res.Image = append(res.Image, dockerImg)
			}
		})
	}

	doc.parseBaseExecutor(&res.BaseExecutor, nameNode, valueNode, parseDocker, "docker")
	return res
}

func (doc *YamlDocument) parseDockerImage(imageNode *sitter.Node) ast.DockerImage {
	// imageNode is a block_sequence_item
	dockerImg := ast.DockerImage{}
	blockNode := GetChildOfType(imageNode, "block_node")

	if blockNode == nil { //TODO: deal with errors
		// Can happen if the docker is an alias/anchor
		return dockerImg
	}

	blockMappingNode := GetChildMapping(blockNode)

	if blockMappingNode == nil { //TODO: deal with errors
		return dockerImg
	}

	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {
		case "image":
			dockerImg.Image = ParseDockerImageValue(doc.GetNodeText(valueNode))
			dockerImg.ImageRange = NodeToRange(child)
		case "name":
			dockerImg.Name = doc.GetNodeText(valueNode)
		case "entrypoint":
			dockerImg.Entrypoint = doc.getNodeTextArrayOrText(valueNode)
		case "command":
			dockerImg.Command = doc.getNodeTextArrayOrText(valueNode)
		case "user":
			dockerImg.User = doc.GetNodeText(valueNode)
		case "environment":
			dockerImg.Environment = doc.parseDictionary(GetChildOfType(valueNode, "block_mapping"))
		case "auth":
			dict := doc.parseDictionary(GetChildOfType(valueNode, "block_mapping"))
			dockerImg.Auth = ast.DockerImageAuth{
				Username: dict["username"],
				Password: dict["password"],
			}
		case "aws_auth":
			dict := doc.parseDictionary(GetChildOfType(valueNode, "block_mapping"))
			dockerImg.AwsAuth = ast.DockerImageAWSAuth{
				AWSAccessKeyID:     dict["AWS_ACCESS_KEY_ID"],
				AWSSecretAccessKey: dict["AWS_SECRET_ACCESS_KEY"],
			}
		}
	})

	return dockerImg
}

func (doc *YamlDocument) parseExecutorRef(valueNode *sitter.Node, child *sitter.Node) (string, protocol.Range, map[string]ast.ParameterValue) {
	executorParameters := map[string]ast.ParameterValue{}
	if valueNode == nil {
		childRange := NodeToRange(child)
		return "", protocol.Range{
			Start: protocol.Position{
				Line:      childRange.Start.Line,
				Character: childRange.Start.Character + uint32(len("executor:")),
			},
			End: protocol.Position{
				Line: childRange.Start.Line,
				// We add 999 to cover the whole line
				Character: childRange.Start.Character + uint32(len("executor:")) + 999,
			},
		}, executorParameters
	}

	// valueNode is either a flow_node or a block_node containing a block_mapping_pair
	if valueNode.Type() == "flow_node" {
		return doc.GetNodeText(valueNode), NodeToRange(child), executorParameters
	}

	name := ""
	blockMapping := GetChildMapping(valueNode)
	if blockMapping == nil { //TODO: deal with errors
		return "", protocol.Range{}, executorParameters
	}

	iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {
		case "name":
			name = doc.GetNodeText(valueNode)
		default:
			value, err := doc.parseParameterValue(child)
			if err == nil {
				executorParameters[keyName] = value
			}
		}
	})

	return name, NodeToRange(child), executorParameters
}

func (doc *YamlDocument) addedMachineTrueDeprecatedDiag(child *sitter.Node) bool {
	_, valueNode := getKeyValueNodes(child)

	value := doc.GetNodeText(valueNode)
	if utils.IsValidYAMLBooleanValue(value) && utils.GetYAMLBooleanValue(value) {
		doc.addDiagnostic(
			protocol.Diagnostic{
				Severity: protocol.DiagnosticSeverityWarning,
				Range:    NodeToRange(child),
				Message:  "Using `machine: true` is deprecated, please instead specify an image to use.",
				Tags: []protocol.DiagnosticTag{
					protocol.DiagnosticTagDeprecated,
				},
			},
		)
		return true
	}

	return false
}

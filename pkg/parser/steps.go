package parser

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (doc *YamlDocument) parseSteps(stepsNode *sitter.Node) []ast.Step {
	// stepsNode is a block_node
	steps := make([]ast.Step, 0)

	blockSequenceNode := GetChildOfType(stepsNode, "block_sequence")
	iterateOnBlockSequence(blockSequenceNode, func(child *sitter.Node) {
		if child.Type() == "block_sequence_item" {
			steps = append(steps, doc.parseSingleStep(child)...)
		}
	})

	return steps
}

func (doc *YamlDocument) parseSingleStep(stepNode *sitter.Node) []ast.Step {
	// stepNode is a block_sequence_item
	if stepNode == nil || stepNode.Type() != "block_sequence_item" {
		// stepNode should be a block_sequence_item with 2 children: `-` and another node
		return []ast.Step{}
	}

	if stepNode.ChildCount() == 1 {
		return []ast.Step{ast.NamedStep{
			Name: "",
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      stepNode.StartPoint().Row,
					Character: stepNode.StartPoint().Column + 1,
				},
				End: protocol.Position{
					Line:      stepNode.StartPoint().Row,
					Character: stepNode.StartPoint().Column + 2,
				},
			},
		}}
	}

	child := stepNode.Child(1) // Either flow_node or block_node

	if child == nil {
		// TODO: Handle error
		return nil
	}

	switch child.Type() {
	case "flow_node":
		return []ast.Step{ast.NamedStep{Name: doc.GetNodeText(child), Range: NodeToRange(child)}}
	case "block_node":
		blockMapping := GetChildOfType(child, "block_mapping")
		if blockMapping == nil {
			return []ast.Step{ast.Run{Range: NodeToRange(child)}}
		}
		blockMappingPair := GetChildOfType(blockMapping, "block_mapping_pair")
		key := blockMappingPair.ChildByFieldName("key")
		keyName := doc.GetNodeText(key)
		valueNode := blockMappingPair.ChildByFieldName("value")
		if valueNode == nil {
			return nil
		}
		switch keyName {
		case "run":
			return []ast.Step{doc.parseRunStep(valueNode)}
		case "checkout":
			return []ast.Step{doc.parseCheckoutStep(valueNode)}
		case "setup_remote_docker":
			return []ast.Step{doc.parseSetupRemoteDockerStep(valueNode)}
		case "save_cache":
			return []ast.Step{doc.parseSaveCacheStep(valueNode)}
		case "restore_cache":
			return []ast.Step{doc.parseRestoreCacheStep(valueNode)}
		case "store_artifacts":
			return []ast.Step{doc.parseStoreArtifactsStep(valueNode)}
		case "store_test_results":
			return []ast.Step{doc.parseStoreTestResultsStep(valueNode)}
		case "persist_to_workspace":
			return []ast.Step{doc.parsePersistToWorkspaceStep(valueNode)}
		case "attach_workspace":
			return []ast.Step{doc.parseAttachWorkspaceStep(valueNode)}
		case "add_ssh_keys":
			return []ast.Step{doc.parseAddSSHKeyStep(valueNode)}
		case "when":
			return doc.parseWhenUnlessStep(valueNode)
		case "unless":
			return doc.parseWhenUnlessStep(valueNode)
		case "steps":
			stepName := doc.GetNodeText(valueNode)
			_, stepName = utils.ExtractParameterName(stepName)
			return []ast.Step{ast.Steps{Name: stepName, Range: NodeToRange(valueNode)}}
		default:
			return []ast.Step{doc.parseNamedStepWithParameters(keyName, valueNode)}
		}
	}

	return []ast.Step{ast.NamedStep{}} // TODO: return error
}

func (doc *YamlDocument) parseWhenUnlessStep(blockNode *sitter.Node) []ast.Step {
	// blockNode is a block_node
	steps := make([]ast.Step, 0)

	blockMapping := GetChildMapping(blockNode)
	iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		key, value := getKeyValueNodes(child)
		if key == nil || value == nil {
			return
		}

		switch doc.GetNodeText(key) {
		case "steps":
			steps = append(steps, doc.parseSteps(value)...)
		}
	})

	return steps
}

func (doc *YamlDocument) parseNamedStepWithParameters(stepName string, namedStepWithParams *sitter.Node) ast.NamedStep {
	// namedStepWithParams is either flow_node or block_node
	if namedStepWithParams == nil {
		return ast.NamedStep{}
	}
	if namedStepWithParams.Type() == "flow_node" {
		return ast.NamedStep{Name: stepName, Range: NodeToRange(namedStepWithParams.Parent().ChildByFieldName("key"))}
	} else { // block_node
		blockMappingNode := GetChildMapping(namedStepWithParams)
		paramRange := protocol.Range{}

		if blockMappingNode != nil {
			paramRange = NodeToRange(blockMappingNode)
		}

		res := ast.NamedStep{
			Name:            stepName,
			Parameters:      make(map[string]ast.ParameterValue),
			Range:           NodeToRange(namedStepWithParams.Parent().ChildByFieldName("key")),
			ParametersRange: paramRange,
		}
		iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			if child != nil {
				keyName := doc.GetNodeText(child.ChildByFieldName("key"))

				if keyName == "" {
					return
				}

				paramValue, err := doc.parseParameterValue(child)
				if err != nil {
					return
				}
				res.Parameters[keyName] = paramValue
			}
		})
		return res
	}
}

func (doc *YamlDocument) parseRunStep(runNode *sitter.Node) ast.Run {
	// runNode is either flow_node or block_node
	if runNode.Type() == "flow_node" {
		commandString := doc.GetNodeText(runNode)
		return ast.Run{Command: commandString, Range: NodeToRange(runNode.Parent().ChildByFieldName("key"))}
	} else { // block_node
		blockScalarNode := GetChildOfType(runNode, "block_scalar")
		if blockScalarNode != nil {
			// This happens when the command in the run step is defined like this:
			// - run: |
			//   	echo "Hello World"

			commandString := doc.GetNodeText(blockScalarNode)
			return ast.Run{Command: commandString}
		}

		blockMappingNode := GetChildMapping(runNode)
		res := ast.Run{Range: NodeToRange(runNode.Parent().ChildByFieldName("key"))}
		iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			keyNode, valueNode := getKeyValueNodes(child)
			if keyNode == nil || valueNode == nil {
				return
			}
			keyName := doc.GetNodeText(keyNode)

			switch keyName {
			case "name":
				res.Name = doc.GetNodeText(valueNode)
			case "command":
				res.CommandRange = NodeToRange(valueNode)
				res.Command = doc.GetNodeText(valueNode)
				res.RawCommand = doc.GetRawNodeText(valueNode)
			case "shell":
				res.Shell = doc.GetNodeText(valueNode)
			case "background":
				res.Background = (doc.GetNodeText(valueNode) == "true")
			case "working_directory":
				res.WorkingDirectory = doc.GetNodeText(valueNode)
			case "no_output_timeout":
				res.NoOutputTimeout = doc.GetNodeText(valueNode)
			case "when":
				res.When = doc.GetNodeText(valueNode)
			case "environment":
				res.Environment = doc.parseDictionary(valueNode)
			}
		})
		return res
	}
}

func (doc *YamlDocument) parseCheckoutStep(checkoutNode *sitter.Node) ast.Checkout {
	// checkoutNode is either flow_node or block_node
	res := ast.Checkout{Path: ".", Range: NodeToRange(checkoutNode.Parent().ChildByFieldName("key"))}
	if checkoutNode.Type() == "flow_node" {
		return res
	} else { // block_node
		blockMappingNode := GetChildMapping(checkoutNode)
		iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			keyNode, valueNode := getKeyValueNodes(child)
			if keyNode == nil || valueNode == nil {
				return
			}
			keyName := doc.GetNodeText(keyNode)
			switch keyName {
			case "path":
				res.Path = doc.GetNodeText(valueNode)
			}
		})
		return res
	}
}

func (doc *YamlDocument) parseSetupRemoteDockerStep(setupRemoteDockerNode *sitter.Node) ast.SetupRemoteDocker {
	// setupRemoteDockerNode is either flow_node or block_node
	res := ast.SetupRemoteDocker{DockerLayerCaching: false, Range: NodeToRange(setupRemoteDockerNode.Parent().ChildByFieldName("key"))}
	if setupRemoteDockerNode.Type() == "flow_node" {
		return res
	} else { // block_node
		blockMappingNode := GetChildMapping(setupRemoteDockerNode)
		iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			keyNode, valueNode := getKeyValueNodes(child)
			if keyNode == nil || valueNode == nil {
				return
			}
			keyName := doc.GetNodeText(keyNode)
			switch keyName {
			case "docker_layer_caching":
				res.DockerLayerCaching = (doc.GetNodeText(valueNode) == "true")
			case "version":
				res.Version = doc.GetNodeText(valueNode)
			}
		})
		return res
	}
}

func (doc *YamlDocument) parseSaveCacheStep(saveCacheNode *sitter.Node) ast.SaveCache {
	blockMappingNode := GetChildMapping(saveCacheNode)
	res := ast.SaveCache{Range: NodeToRange(saveCacheNode.Parent().ChildByFieldName("key"))}
	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := getKeyValueNodes(child)
		if keyNode == nil || valueNode == nil {
			return
		}
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "paths":
			res.Paths = doc.getNodeTextArray(valueNode)
		case "key":
			res.Key = doc.GetNodeText(valueNode)
		case "name":
			res.CacheName = doc.GetNodeText(valueNode)
		}
	})

	return res
}

func (doc *YamlDocument) parseRestoreCacheStep(restoreCacheNode *sitter.Node) ast.RestoreCache {
	blockMappingNode := GetChildMapping(restoreCacheNode)
	res := ast.RestoreCache{Range: NodeToRange(restoreCacheNode.Parent().ChildByFieldName("key"))}
	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := getKeyValueNodes(child)
		if keyNode == nil || valueNode == nil {
			return
		}
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "key":
			res.Key = doc.GetNodeText(valueNode)
		case "keys":
			res.Keys = doc.getNodeTextArray(valueNode)
		case "name":
			res.CacheName = doc.GetNodeText(valueNode)
		}
	})
	return res
}

func (doc *YamlDocument) parseStoreArtifactsStep(storeArtifactsNode *sitter.Node) ast.StoreArtifacts {
	blockMappingNode := GetChildMapping(storeArtifactsNode)
	res := ast.StoreArtifacts{Range: NodeToRange(storeArtifactsNode.Parent().ChildByFieldName("key"))}
	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := getKeyValueNodes(child)
		if keyNode == nil || valueNode == nil {
			return
		}
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "path":
			res.Path = doc.GetNodeText(valueNode)
		case "destination":
			res.Destination = doc.GetNodeText(valueNode)
		}
	})
	return res
}

func (doc *YamlDocument) parseStoreTestResultsStep(storeTestResultsNode *sitter.Node) ast.StoreTestResults {
	blockMappingNode := GetChildMapping(storeTestResultsNode)
	res := ast.StoreTestResults{Range: NodeToRange(storeTestResultsNode.Parent().ChildByFieldName("key"))}
	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := getKeyValueNodes(child)
		if keyNode == nil || valueNode == nil {
			return
		}
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "path":
			res.Path = doc.GetNodeText(valueNode)
		}
	})
	return res
}

func (doc *YamlDocument) parsePersistToWorkspaceStep(persistToWorkspaceNode *sitter.Node) ast.PersistToWorkspace {
	blockMappingNode := GetChildMapping(persistToWorkspaceNode)
	res := ast.PersistToWorkspace{Range: NodeToRange(persistToWorkspaceNode.Parent().ChildByFieldName("key"))}
	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := getKeyValueNodes(child)
		if keyNode == nil || valueNode == nil {
			return
		}
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "root":
			res.Root = doc.GetNodeText(valueNode)
		case "paths":
			res.Paths = doc.getNodeTextArray(valueNode)
		}
	})
	return res
}

func (doc *YamlDocument) parseAttachWorkspaceStep(attachWorkspaceNode *sitter.Node) ast.AttachWorkspace {
	blockMappingNode := GetChildMapping(attachWorkspaceNode)
	res := ast.AttachWorkspace{Range: NodeToRange(attachWorkspaceNode.Parent().ChildByFieldName("key"))}
	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := getKeyValueNodes(child)
		if keyNode == nil || valueNode == nil {
			return
		}
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "at":
			res.At = doc.GetNodeText(valueNode)
		}
	})
	return res
}

func (doc *YamlDocument) parseAddSSHKeyStep(addSSHKeyNode *sitter.Node) ast.AddSSHKey {
	blockMappingNode := GetChildMapping(addSSHKeyNode)
	res := ast.AddSSHKey{Range: NodeToRange(addSSHKeyNode.Parent().ChildByFieldName("key"))}
	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := getKeyValueNodes(child)
		if keyNode == nil || valueNode == nil {
			return
		}
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "fingerprints":
			res.Fingerprints = doc.getNodeTextArray(valueNode)
		}
	})
	return res
}

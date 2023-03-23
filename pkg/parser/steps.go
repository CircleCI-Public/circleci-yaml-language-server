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

	blockSequenceNode := GetChildSequence(stepsNode)
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
		if GetFirstChild(child).Type() != "alias" {
			return []ast.Step{ast.NamedStep{Name: doc.GetNodeText(child), Range: doc.NodeToRange(child)}}
		}
		step := doc.YamlAnchors[doc.GetNodeText(child)[1:]].ValueNode
		if step == nil {
			return nil
		}
		blockMapping := GetChildOfType(step, "block_mapping")
		if blockMapping == nil {
			step = GetChildOfType(step, "plain_scalar")
			if step == nil {
				return nil
			}
			return []ast.Step{ast.NamedStep{Name: doc.GetNodeText(step), Range: doc.NodeToRange(step)}}
		}
		return doc.parseStep(blockMapping)
	case "block_node":
		blockMapping := GetChildOfType(child, "block_mapping")
		if blockMapping == nil {
			return []ast.Step{ast.Run{Range: doc.NodeToRange(child)}}
		}
		return doc.parseStep(blockMapping)
	}

	return []ast.Step{ast.NamedStep{}} // TODO: return error
}

func (doc *YamlDocument) parseStep(blockMapping *sitter.Node) []ast.Step {
	blockMappingPair := GetChildOfType(blockMapping, "block_mapping_pair")
	keyNode, valueNode := doc.GetKeyValueNodes(blockMappingPair)
	keyName := doc.GetNodeText(keyNode)
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
		return []ast.Step{ast.Steps{Name: stepName, Range: doc.NodeToRange(valueNode)}}
	case "<<":
		return doc.parseAnchorStep(valueNode)
	default:
		return []ast.Step{doc.parseNamedStepWithParameters(keyName, valueNode)}
	}
}

func (doc *YamlDocument) parseAnchorStep(blockNode *sitter.Node) []ast.Step {
	blockMapping := GetChildOfType(blockNode, "block_mapping")
	blockSequence := GetChildSequence(blockNode)

	if blockSequence != nil {
		blockSequenceItem := GetChildOfType(blockSequence, "block_sequence_item")
		return doc.parseSingleStep(blockSequenceItem)
	}

	if blockMapping != nil {
		return doc.parseStep(blockMapping)
	}

	return nil
}

func (doc *YamlDocument) parseWhenUnlessStep(blockNode *sitter.Node) []ast.Step {
	// blockNode is a block_node
	steps := make([]ast.Step, 0)

	blockMapping := GetChildMapping(blockNode)
	doc.iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		key, value := doc.GetKeyValueNodes(child)
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
	hasFlowMapping := GetChildOfType(namedStepWithParams, "flow_mapping") != nil
	if namedStepWithParams.Type() == "flow_node" && !hasFlowMapping {
		stepNameNode, _ := doc.GetKeyValueNodes(namedStepWithParams.Parent())
		rng := doc.NodeToRange(stepNameNode)
		return ast.NamedStep{Name: stepName, Range: rng}
	} else { // block_node
		blockMappingNode := GetChildMapping(namedStepWithParams)
		paramRange := protocol.Range{}

		if blockMappingNode != nil {
			paramRange = doc.NodeToRange(blockMappingNode)
		}

		res := ast.NamedStep{
			Name:            stepName,
			Parameters:      make(map[string]ast.ParameterValue),
			Range:           doc.NodeToRange(namedStepWithParams.Parent().ChildByFieldName("key")),
			ParametersRange: paramRange,
		}
		doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			if child == nil {
				return
			}
			key, _ := doc.GetKeyValueNodes(child)
			keyName := doc.GetNodeText(key)

			if keyName == "" {
				return
			}

			paramValue, err := doc.parseParameterValue(child)
			if err != nil {
				return
			}
			res.Parameters[keyName] = paramValue
		})
		return res
	}
}

func (doc *YamlDocument) parseRunStep(runNode *sitter.Node) ast.Run {
	// runNode is either flow_node or block_node
	if runNode.Type() == "flow_node" {
		commandString := doc.GetNodeText(runNode)
		return ast.Run{
			Name:         "run",
			Command:      commandString,
			CommandRange: doc.NodeToRange(runNode),
			Range:        doc.NodeToRange(runNode.Parent().ChildByFieldName("key")),
			RawCommand:   doc.GetRawNodeText(runNode),
		}
	} else { // block_node
		blockScalarNode := GetChildOfType(runNode, "block_scalar")
		if blockScalarNode != nil {
			// This happens when the command in the run step is defined like this:
			// - run: |
			//   	echo "Hello World"

			commandString := doc.GetNodeText(blockScalarNode)
			return ast.Run{
				Name:         "run",
				Command:      commandString,
				CommandRange: doc.NodeToRange(blockScalarNode),
				Range:        doc.NodeToRange(runNode.Parent().ChildByFieldName("key")),
				RawCommand:   doc.GetRawNodeText(blockScalarNode),
			}
		}

		blockMappingNode := GetChildMapping(runNode)
		res := ast.Run{Name: "run", Range: doc.NodeToRange(runNode.Parent().ChildByFieldName("key"))}
		doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			keyNode, valueNode := doc.GetKeyValueNodes(child)
			if keyNode == nil || valueNode == nil {
				return
			}
			keyName := doc.GetNodeText(keyNode)

			switch keyName {
			case "name":
				res.Name = doc.GetNodeText(valueNode)
			case "command":
				res.CommandRange = doc.NodeToRange(valueNode)
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
				res.WhenRange = doc.NodeToRange(valueNode)
			case "environment":
				res.Environment = doc.parseDictionary(valueNode)
			}
		})
		return res
	}
}

func (doc *YamlDocument) parseCheckoutStep(checkoutNode *sitter.Node) ast.Checkout {
	// checkoutNode is either flow_node or block_node
	res := ast.Checkout{Path: ".", Range: doc.NodeToRange(checkoutNode.Parent().ChildByFieldName("key"))}
	if checkoutNode.Type() == "flow_node" {
		return res
	} else { // block_node
		blockMappingNode := GetChildMapping(checkoutNode)
		doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			keyNode, valueNode := doc.GetKeyValueNodes(child)
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
	res := ast.SetupRemoteDocker{DockerLayerCaching: false, Range: doc.NodeToRange(setupRemoteDockerNode.Parent().ChildByFieldName("key"))}
	if setupRemoteDockerNode.Type() == "flow_node" {
		return res
	} else { // block_node
		blockMappingNode := GetChildMapping(setupRemoteDockerNode)
		doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			keyNode, valueNode := doc.GetKeyValueNodes(child)
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
	res := ast.SaveCache{Range: doc.NodeToRange(saveCacheNode.Parent().ChildByFieldName("key"))}
	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
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
	res := ast.RestoreCache{Range: doc.NodeToRange(restoreCacheNode.Parent().ChildByFieldName("key"))}
	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
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
	res := ast.StoreArtifacts{Range: doc.NodeToRange(storeArtifactsNode.Parent().ChildByFieldName("key"))}
	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
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
	res := ast.StoreTestResults{Range: doc.NodeToRange(storeTestResultsNode.Parent().ChildByFieldName("key"))}
	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
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
	res := ast.PersistToWorkspace{Range: doc.NodeToRange(persistToWorkspaceNode.Parent().ChildByFieldName("key"))}
	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
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
	res := ast.AttachWorkspace{Range: doc.NodeToRange(attachWorkspaceNode.Parent().ChildByFieldName("key"))}
	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
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
	res := ast.AddSSHKey{Range: doc.NodeToRange(addSSHKeyNode.Parent().ChildByFieldName("key"))}
	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
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

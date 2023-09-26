package languageservice

import (
	"regexp"
	"sort"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

type Tokens struct {
	Position       protocol.Position
	Length         uint32
	TokenType      uint32
	TokenModifiers uint32
}

type SemanticTokenStruct struct {
	prev            *[]uint32
	processedTokens *[]uint32
	doc             parser.YamlDocument
	tokens          *[]Tokens
}

var PARAM_REGEX, _ = regexp.Compile(`<<\s*(parameters|pipeline.parameters)\.([A-z0-9-_]*)\s*>>`)

func SemanticTokens(params protocol.SemanticTokensParams, cache *utils.Cache, context *utils.LsContext) protocol.SemanticTokens {
	doc, err := parser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)

	if err != nil {
		return protocol.SemanticTokens{}
	}

	semanticTokens := SemanticTokenStruct{
		prev:            &[]uint32{0, 0},
		processedTokens: &[]uint32{},
		tokens:          &[]Tokens{},
		doc:             doc,
	}

	iter := sitter.NewIterator(doc.RootNode, sitter.DFSMode)
	iter.ForEach(func(node *sitter.Node) error {
		if node.Type() == "block_mapping_pair" {
			keyNode, valueNode := doc.GetKeyValueNodes(node)

			if keyNode != nil {
				semanticTokens.highlightOrbs(keyNode)
				semanticTokens.highlightBuiltInKeywords(keyNode)
			}

			if valueNode != nil {
				semanticTokens.highlightParameters(valueNode)
				semanticTokens.highlightCacheKeys(valueNode)
				semanticTokens.highlightOrbs(valueNode)
			}
		}

		if node.Type() == "block_sequence_item" {
			if child := parser.GetChildOfType(node, "flow_node"); child != nil {
				semanticTokens.highlightCacheKeys(child)
				semanticTokens.highlightOrbs(child)
				semanticTokens.highlightParameters(child)
			}
		}

		return nil
	})

	for _, command := range doc.Commands {
		semanticTokens.highlightSteps(command.Steps)
	}

	for _, jobs := range doc.Jobs {
		semanticTokens.highlightSteps(jobs.Steps)
	}

	semanticTokens.processTokens()

	return protocol.SemanticTokens{
		Data: *semanticTokens.processedTokens,
	}
}

var KEYWORDS = []string{
	"parameters", "description", "executor", "steps", "filters",
	"environment", "working_directory", "docker", "requires", "jobs", "triggers",
}

var ROOT_KEYWORDS = []string{
	"workflows", "jobs", "version", "commands",
	"executors", "parameters", "orbs", "setup",
}

func (sem SemanticTokenStruct) highlightBuiltInKeywords(keyNode *sitter.Node) {
	if keyName := sem.doc.GetNodeText(keyNode); keyNode.Type() == "flow_node" && utils.FindInArray(KEYWORDS, keyName) != -1 {
		length := keyNode.EndPoint().Column - keyNode.StartPoint().Column
		sem.addToken(protocol.Position{Line: keyNode.StartPoint().Row, Character: keyNode.StartPoint().Column}, length, 0, 0)
	}

	// Needed in order to make sure we are at the top level of the YAML file
	blockMappingPair := keyNode.Parent()
	if blockMappingPair == nil {
		return
	}
	blockMapping := blockMappingPair.Parent()
	if blockMapping == nil {
		return
	}
	blockNode := blockMapping.Parent()
	if blockNode == nil {
		return
	}
	document := blockNode.Parent()
	if document == nil {
		return
	}

	if keyName := sem.doc.GetNodeText(keyNode); document.Type() == "document" && keyNode.Type() == "flow_node" && utils.FindInArray(ROOT_KEYWORDS, keyName) != -1 {
		length := keyNode.EndPoint().Column - keyNode.StartPoint().Column
		sem.addToken(protocol.Position{Line: keyNode.StartPoint().Row, Character: keyNode.StartPoint().Column}, length, 0, 0)
	}
}

func (sem SemanticTokenStruct) highlightParameters(valueNode *sitter.Node) {
	sem.highlightWithRegex(valueNode, PARAM_REGEX)
}

func (sem SemanticTokenStruct) highlightCacheKeys(valueNode *sitter.Node) {
	reg, err := regexp.Compile(`{{ ?(.Branch|.BuildNum|.Revision|.CheckoutKey|.Environment.variableName|checksum .*|epoch|arch) ?}}`)

	if err != nil {
		return
	}

	sem.highlightWithRegex(valueNode, reg)
}

func (sem SemanticTokenStruct) highlightOrbs(valueNode *sitter.Node) {
	if valueNode.Type() == "flow_node" {
		content := sem.doc.GetRawNodeText(valueNode)
		if sem.doc.IsOrbReference(content) {
			// Orb method
			slashIdx := strings.Index(content, "/")
			if slashIdx == -1 {
				// Should never happen
				return
			}

			orbMethodLength := valueNode.EndPoint().Column - valueNode.StartPoint().Column - uint32(slashIdx) - 1
			orbNameLength := uint32(slashIdx) + 1 // +1 for the slash

			// Highlight orb name
			sem.addToken(protocol.Position{Line: valueNode.StartPoint().Row, Character: valueNode.StartPoint().Column}, orbNameLength, 1, 0)

			// Highlight orb method
			sem.addToken(protocol.Position{Line: valueNode.StartPoint().Row, Character: valueNode.StartPoint().Column + orbNameLength}, orbMethodLength, 0, 0)
		} else if _, ok := sem.doc.Orbs[content]; ok && utils.PosInRange(sem.doc.OrbsRange, sem.doc.NodeToRange(valueNode).Start) {
			// Orb definition in the orbs section
			rng := sem.doc.NodeToRange(valueNode)
			sem.addToken(rng.Start, rng.End.Character-rng.Start.Character, 1, 0)
		}
	}
}

func (sem SemanticTokenStruct) highlightWithRegex(valueNode *sitter.Node, regex *regexp.Regexp) {
	child := parser.GetFirstChild(valueNode)
	isFlowNode := valueNode.Type() == "flow_node"
	isBlockScalar := valueNode.Type() == "block_node" && child != nil && child.Type() == "block_scalar"

	if !isFlowNode && !isBlockScalar {
		return
	}

	content := sem.doc.GetRawNodeText(valueNode)
	params := regex.FindAllIndex([]byte(content), -1)

	for _, param := range params {
		length := param[1] - param[0]

		if length < 0 {
			continue
		}

		startPos := utils.IndexToPos(param[0], []byte(content))
		startPos.Line += valueNode.StartPoint().Row

		if isFlowNode {
			startPos.Character += valueNode.StartPoint().Column
		}

		sem.addToken(startPos, uint32(length), 0, 0)
	}
}

func (sem SemanticTokenStruct) highlightSteps(steps []ast.Step) {
	for _, step := range steps {
		sem.highlightStep(step)
	}
}

func (sem SemanticTokenStruct) highlightStep(step ast.Step) {
	switch step := step.(type) {
	case ast.Run:
		sem.highlightCommand(step.RawCommand, step.CommandRange)
	}
}

func (sem SemanticTokenStruct) highlightCommand(rawCommand string, commandRange protocol.Range) {
	// To improve readability, commands should be higlighted in a different color than parameters,
	// having two semantics on the same range doesn't work and only one is kept (probably the longest ranging one)
	// to be sure parameters are correctly highlighted
	// command highlighting should not interfere with the sem.highlightParemeters function
	// and only be inserted in between parameters
	parts := strings.Split(rawCommand, "\n")
	baseOffset := commandRange.Start.Character

	if len(parts) > 1 {
		// Means we are in a block_scalar -> baseOffset should be set to 0
		baseOffset = 0
	}

	for i, cmd := range parts {
		offset := uint32(0)

		if strings.HasPrefix(strings.TrimSpace(cmd), "#") {
			sem.addTokenRange(
				protocol.Range{
					Start: protocol.Position{
						Line:      commandRange.Start.Line + uint32(i),
						Character: baseOffset,
					},
					End: protocol.Position{
						Line:      commandRange.Start.Line + uint32(i),
						Character: uint32(len(cmd)) + baseOffset,
					},
				},
				3,
				0,
			)

			continue
		}

		// Find parameters match indexes to add tokens on ranges in between
		matches := PARAM_REGEX.FindAllIndex([]byte(cmd), -1)

		// Filling an additional (fake) match to reach end of line
		matches = append(
			matches,
			[]int{len(cmd), len(cmd)},
		)

		for _, matchIndexes := range matches {
			sem.addTokenRange(
				protocol.Range{
					Start: protocol.Position{
						Line:      commandRange.Start.Line + uint32(i),
						Character: offset + baseOffset,
					},
					End: protocol.Position{
						Line:      commandRange.Start.Line + uint32(i),
						Character: uint32(matchIndexes[0]) + baseOffset,
					},
				},
				4,
				0,
			)

			offset = uint32(matchIndexes[1])
		}
	}
}

// Because it's not very well optimized, use this function only if you're not sure that the element is on a single line
func (sem SemanticTokenStruct) addTokenRange(rng protocol.Range, tokenType uint32, tokenModifiers uint32) {
	startIdx := utils.PosToIndex(rng.Start, sem.doc.Content)
	endIdx := utils.PosToIndex(rng.End, sem.doc.Content)

	sem.addToken(rng.Start, uint32(endIdx-startIdx), tokenType, tokenModifiers)
}

func (sem SemanticTokenStruct) addToken(pos protocol.Position, length uint32, tokenType uint32, tokenModifiers uint32) {
	*sem.tokens = append(*sem.tokens, Tokens{pos, length, tokenType, tokenModifiers})
}

func (sem SemanticTokenStruct) processTokens() {
	sort.SliceStable(*sem.tokens, func(i, j int) bool {
		if (*sem.tokens)[i].Position.Line == (*sem.tokens)[j].Position.Line {
			return (*sem.tokens)[i].Position.Character < (*sem.tokens)[j].Position.Character
		}
		return (*sem.tokens)[i].Position.Line < (*sem.tokens)[j].Position.Line
	})

	for _, token := range *sem.tokens {
		sem.processToken(token)
	}
}

func (sem SemanticTokenStruct) processToken(token Tokens) {
	*sem.processedTokens = append(*sem.processedTokens, token.Position.Line-(*sem.prev)[0])

	if token.Position.Line == (*sem.prev)[0] {
		*sem.processedTokens = append(*sem.processedTokens, token.Position.Character-(*sem.prev)[1])
	} else {
		*sem.processedTokens = append(*sem.processedTokens, token.Position.Character)
	}

	*sem.processedTokens = append(*sem.processedTokens, token.Length)
	*sem.processedTokens = append(*sem.processedTokens, token.TokenType)
	*sem.processedTokens = append(*sem.processedTokens, token.TokenModifiers)

	*(sem.prev) = []uint32{token.Position.Line, token.Position.Character}
}

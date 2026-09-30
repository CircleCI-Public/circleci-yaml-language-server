package languageservice

import (
	"regexp"
	"slices"
	"sort"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	parser2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
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
	doc             parser2.YamlDocument
	lines           position.Lines
	tokens          *[]Tokens
}

func SemanticTokens(params protocol.SemanticTokensParams, cache *cache.Cache, context *session.Settings) protocol.SemanticTokens {
	doc, err := parser2.ParseFromUriWithCache(params.TextDocument.URI, cache, context)
	if err != nil {
		return protocol.SemanticTokens{}
	}
	defer doc.Close()

	semanticTokens := SemanticTokenStruct{
		prev:            &[]uint32{0, 0},
		processedTokens: &[]uint32{},
		tokens:          &[]Tokens{},
		doc:             doc,
		lines:           position.NewLines(doc.Content),
	}

	for node := range yamltree.Walk(doc.RootNode) {
		if node.Kind() == "block_mapping_pair" {
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

		if node.Kind() == "block_sequence_item" {
			if child := parser2.GetChildOfType(node, "flow_node"); child != nil {
				semanticTokens.highlightCacheKeys(child)
				semanticTokens.highlightOrbs(child)
				semanticTokens.highlightParameters(child)
			}
		}
	}

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
	"workflows", "jobs", "job-groups", "version", "commands",
	"executors", "parameters", "orbs", "setup",
}

func (sem SemanticTokenStruct) highlightBuiltInKeywords(keyNode *sitter.Node) {
	keyName := sem.doc.GetNodeText(keyNode)
	if keyNode.Kind() != "flow_node" {
		return
	}
	if slices.Contains(KEYWORDS, keyName) {
		length := position.End(keyNode).Character - position.Start(keyNode).Character
		sem.addToken(protocol.Position{Line: position.Start(keyNode).Line, Character: position.Start(keyNode).Character}, length, 0, 0)
		return
	}
	// Only a root keyword is worth reading up the tree for.
	if !slices.Contains(ROOT_KEYWORDS, keyName) {
		return
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

	if document.Kind() == "document" {
		length := position.End(keyNode).Character - position.Start(keyNode).Character
		sem.addToken(protocol.Position{Line: position.Start(keyNode).Line, Character: position.Start(keyNode).Character}, length, 0, 0)
	}
}

func (sem SemanticTokenStruct) highlightParameters(valueNode *sitter.Node) {
	sem.highlightWithRegex(valueNode, paramref.Pattern)
}

var cacheKeyTemplateRegex = regexp.MustCompile(`{{ ?(.Branch|.BuildNum|.Revision|.CheckoutKey|.Environment.variableName|checksum .*|epoch|arch) ?}}`)

func (sem SemanticTokenStruct) highlightCacheKeys(valueNode *sitter.Node) {
	sem.highlightWithRegex(valueNode, cacheKeyTemplateRegex)
}

func (sem SemanticTokenStruct) highlightOrbs(valueNode *sitter.Node) {
	if valueNode.Kind() != "flow_node" {
		return
	}

	content := sem.doc.ScalarText(valueNode)
	start := position.Start(valueNode)
	if child := parser2.GetFirstChild(valueNode); child != nil && (child.Kind() == "double_quote_scalar" || child.Kind() == "single_quote_scalar") {
		start.Character++
	}
	orbs, orbsRange := sem.orbsInScope(start)

	if orbName, method, ok := strings.Cut(content, "/"); ok && !strings.Contains(method, "/") {
		if _, declared := orbs[orbName]; !declared {
			return
		}
		orbNameLength := uint32(len(orbName)) + 1 // +1 for the slash

		sem.addToken(start, orbNameLength, 1, 0)
		sem.addToken(protocol.Position{Line: start.Line, Character: start.Character + orbNameLength}, uint32(len(method)), 0, 0)
	} else if _, ok := orbs[content]; ok && position.InRange(orbsRange, start) {
		// Orb definition in the orbs section
		sem.addToken(start, uint32(len(content)), 1, 0)
	}
}

// orbsInScope returns the orbs that can be named at pos, and the range of the
// section declaring them: an inline orb's own orbs inside it, and the config's
// elsewhere.
func (sem SemanticTokenStruct) orbsInScope(pos protocol.Position) (map[string]ast.Orb, protocol.Range) {
	orbs, orbsRange, inlineOrbs := sem.doc.Orbs, sem.doc.OrbsRange, sem.doc.LocalOrbInfo

	for {
		var inner *ast.OrbInfo
		for name, orb := range orbs {
			if orb.Url.IsLocal && position.InRange(orb.ValueRange, pos) {
				inner = inlineOrbs[name]
				break
			}
		}
		if inner == nil {
			return orbs, orbsRange
		}
		orbs, orbsRange, inlineOrbs = inner.Orbs, inner.OrbsRange, inner.LocalOrbInfo
	}
}

func (sem SemanticTokenStruct) highlightWithRegex(valueNode *sitter.Node, regex *regexp.Regexp) {
	child := parser2.GetFirstChild(valueNode)
	isFlowNode := valueNode.Kind() == "flow_node"
	isBlockScalar := valueNode.Kind() == "block_node" && child != nil && child.Kind() == "block_scalar"

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

		startPos := position.FromIndex(param[0], []byte(content))
		startPos.Line += position.Start(valueNode).Line

		if isFlowNode {
			startPos.Character += position.Start(valueNode).Character
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
		matches := paramref.Pattern.FindAllIndex([]byte(cmd), -1)

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

func (sem SemanticTokenStruct) addTokenRange(rng protocol.Range, tokenType uint32, tokenModifiers uint32) {
	startIdx := sem.lines.ToIndex(rng.Start)
	endIdx := sem.lines.ToIndex(rng.End)

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

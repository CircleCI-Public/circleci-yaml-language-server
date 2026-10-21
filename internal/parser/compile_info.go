package parser

import (
	"strings"

	sitter "github.com/odvcencio/gotreesitter"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

// CompileInfo holds what the compiler needs and the editor doesn't. It is
// keyed by ranges rather than nodes, so it outlives Close.
type CompileInfo struct {
	// Source is the config's path, or the orb's reference.
	Source string

	// Steps are the step lists as written, keyed by the range of the steps
	// value. A list given as an argument, to a step or a job invocation, is
	// keyed by its ParameterValue's ValueRange, its bare names read as steps.
	Steps        map[protocol.Range][]StepNode
	JobSteps     map[string]protocol.Range
	CommandSteps map[string]protocol.Range
	// PreSteps and PostSteps are keyed by the JobInvocationRange.
	PreSteps  map[protocol.Range]protocol.Range
	PostSteps map[protocol.Range]protocol.Range

	// Filters are keyed by the JobInvocationRange.
	Filters            map[protocol.Range]ast.ParameterValue
	WorkflowConditions map[string][]Condition

	// Orbs are the inline orbs.
	Orbs map[string]*CompileInfo

	// frames are the step lists being read, innermost last.
	frames [][]StepNode
	// pending is what the step just read adds to its list in place of its
	// flat steps: a Condition, or the nodes of a nested list.
	pending []StepNode
	// resolving are the anchors whose values are being read, so that an
	// anchor holding an alias of itself isn't read forever.
	resolving map[string]bool
}

// StepNode is a step in a list as written.
type StepNode struct {
	// Step is set for a plain step.
	Step ast.Step
	// Cond is set for when, unless and with_tool_cache.
	Cond *Condition
}

// Condition is a when, unless or with_tool_cache, with its condition as
// written: a scalar, or a logic statement.
type Condition struct {
	Kind      string
	Condition ast.ParameterValue
	Steps     []StepNode
	Range     protocol.Range
}

func newCompileInfo(source string) *CompileInfo {
	return &CompileInfo{
		Source:             source,
		Steps:              map[protocol.Range][]StepNode{},
		JobSteps:           map[string]protocol.Range{},
		CommandSteps:       map[string]protocol.Range{},
		PreSteps:           map[protocol.Range]protocol.Range{},
		PostSteps:          map[protocol.Range]protocol.Range{},
		Filters:            map[protocol.Range]ast.ParameterValue{},
		WorkflowConditions: map[string][]Condition{},
		Orbs:               map[string]*CompileInfo{},
		resolving:          map[string]bool{},
	}
}

// ParseForCompile parses a config as ParseFromContent does, and also records
// its CompileInfo.
func ParseForCompile(content []byte, source string, context *session.Settings) (YamlDocument, error) {
	return parseForCompile(content, source, context, "", protocol.Position{})
}

func parseForCompile(content []byte, source string, context *session.Settings, URI uri.URI, offset protocol.Position) (YamlDocument, error) {
	doc := ParseFile(content, context)
	doc.URI = URI
	doc.compile = newCompileInfo(source)

	doc.ParseYAML(context, offset)

	return doc, nil
}

// Compile is the document's CompileInfo, nil unless it was parsed by
// ParseForCompile.
func (doc *YamlDocument) Compile() *CompileInfo {
	return doc.compile
}

func (c *CompileInfo) openSteps() {
	c.frames = append(c.frames, []StepNode{})
}

// addSteps adds the step just read, whose flat steps are flat, to the list
// being read.
func (c *CompileInfo) addSteps(flat []ast.Step) {
	top := &c.frames[len(c.frames)-1]
	if c.pending != nil {
		*top = append(*top, c.pending...)
		c.pending = nil
		return
	}
	for _, step := range flat {
		*top = append(*top, StepNode{Step: step})
	}
}

// closeSteps records the list just read, and leaves it pending for the list
// holding it, if any.
func (c *CompileInfo) closeSteps(rng protocol.Range) {
	nodes := c.frames[len(c.frames)-1]
	c.frames = c.frames[:len(c.frames)-1]
	c.Steps[rng] = nodes
	c.pending = nodes
}

func (c *CompileInfo) dropPending() {
	c.pending = nil
}

// compileValue reads the value of pair as written. Scalars are read by
// parseParameterValue. Mappings and sequences are walked here, since
// parseParameterValue reads a sequence of mappings as steps, and aliases are
// resolved, since the value must hold no nodes once the tree is closed.
func (doc *YamlDocument) compileValue(pair *sitter.Node) ast.ParameterValue {
	key, value := doc.GetKeyValueNodes(pair)
	return doc.compileNodeValue(pair, doc.GetNodeText(key), value, doc.NodeToRange(pair))
}

// compileNodeValue reads node, the value of pair if pair isn't nil.
func (doc *YamlDocument) compileNodeValue(pair *sitter.Node, name string, node *sitter.Node, rng protocol.Range) ast.ParameterValue {
	res := ast.ParameterValue{Name: name, Range: rng, ValueRange: doc.NodeToRange(node)}
	first := GetFirstChild(node)
	if first == nil {
		return res
	}

	switch first.Type(yamltree.Language) {
	case "alias":
		anchorName := strings.TrimPrefix(doc.GetNodeText(first), "*")
		anchor, ok := doc.YamlAnchors[anchorName]
		if !ok || doc.compile.resolving[anchorName] {
			return res
		}
		doc.compile.resolving[anchorName] = true
		defer delete(doc.compile.resolving, anchorName)
		return doc.compileNodeValue(nil, name, anchor.ValueNode, rng)

	case "block_mapping", "flow_mapping":
		value := map[string]ast.ParameterValue{}
		doc.iterateOnBlockMapping(first, func(child *sitter.Node) {
			key, valueNode := doc.GetKeyValueNodes(child)
			if key == nil || valueNode == nil {
				return
			}
			value[doc.GetNodeText(key)] = doc.compileNodeValue(child, doc.GetNodeText(key), valueNode, doc.NodeToRange(child))
		})
		res.Value, res.Type = value, "map"
		return res

	case "block_sequence", "flow_sequence":
		values := []ast.ParameterValue{}
		iterateOnBlockSequence(first, func(item *sitter.Node) {
			itemNode := item
			switch item.Type(yamltree.Language) {
			case "block_sequence_item":
				if item.ChildCount() < 2 {
					return
				}
				itemNode = item.Child(1)
			case "flow_node":
			default:
				return
			}
			values = append(values, doc.compileNodeValue(nil, name, itemNode, doc.NodeToRange(itemNode)))
		})
		res.Value, res.Type = values, "enum"
		return res
	}

	var value ast.ParameterValue
	var err error
	if pair != nil {
		value, err = doc.parseParameterValue(pair)
	} else {
		value, err = doc.parseSimpleParameterValue(name, node, rng)
	}
	if err != nil {
		res.Value, res.Type = doc.GetNodeText(node), "string"
		return res
	}
	return value
}

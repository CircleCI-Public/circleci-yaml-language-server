package compile

import (
	"strconv"
	"testing"

	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

// value reads src as a value written in a config, as CompileInfo holds it.
func value(t *testing.T, src string) ast.ParameterValue {
	t.Helper()
	var doc yaml.Node
	assert.NilError(t, yaml.Unmarshal([]byte(src), &doc))
	assert.Assert(t, len(doc.Content) == 1, "%q is one value", src)
	return fromNode(t, doc.Content[0])
}

func fromNode(t *testing.T, n *yaml.Node) ast.ParameterValue {
	t.Helper()
	at := protocol.Position{Line: uint32(n.Line - 1), Character: uint32(n.Column - 1)}
	pv := ast.ParameterValue{Range: protocol.Range{Start: at}}
	switch n.Kind {
	case yaml.MappingNode:
		m := map[string]ast.ParameterValue{}
		for i := 0; i < len(n.Content); i += 2 {
			item := fromNode(t, n.Content[i+1])
			item.Range.Start = protocol.Position{Line: uint32(n.Content[i].Line - 1), Character: uint32(n.Content[i].Column - 1)}
			m[n.Content[i].Value] = item
		}
		pv.Value = m
	case yaml.SequenceNode:
		items := []ast.ParameterValue{}
		for _, c := range n.Content {
			items = append(items, fromNode(t, c))
		}
		pv.Value = items
	default:
		switch n.Tag {
		case "!!int":
			i, err := strconv.Atoi(n.Value)
			assert.NilError(t, err)
			pv.Value = i
		case "!!bool":
			pv.Value = n.Value == "true"
		case "!!null":
		default:
			pv.Value = n.Value
		}
	}
	return pv
}

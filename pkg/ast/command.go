package ast

import "go.lsp.dev/protocol"

type Command struct {
	Range            protocol.Range
	NameRange        protocol.Range
	Name             string
	Description      string
	DescriptionRange protocol.Range
	Steps            []Step
	StepsRange       protocol.Range
	Parameters       map[string]Parameter
	ParametersRange  protocol.Range
}

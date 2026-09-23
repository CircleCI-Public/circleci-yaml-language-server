package ast

import (
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"
)

type Orb struct {
	Url          OrbURL
	Name         string
	Range        protocol.Range
	NameRange    protocol.Range
	VersionRange protocol.Range
	ValueRange   protocol.Range
	// ValueNode belongs to the syntax tree of the document the orb was
	// declared in, and is freed with it. It must not be kept past that
	// document: nothing that is cached may hold it.
	ValueNode *sitter.Node
}

type OrbURL struct {
	IsLocal bool
	Name    string
	Version string
}

type OrbURLDefinition struct {
	Namespace TextAndRange
	Name      TextAndRange
	Version   TextAndRange
}

func (orb *OrbURL) GetOrbID() string {
	if orb.IsLocal {
		return orb.Name
	}
	return fmt.Sprintf("%s@%s", orb.Name, orb.Version)
}

type OrbInfo struct {
	OrbParsedAttributes
	IsLocal bool

	CreatedAt   string
	Description string
	Source      string
	RemoteInfo  RemoteOrbInfo
}

type OrbParsedAttributes struct {
	URI  protocol.URI
	Name string

	Commands           map[string]Command
	Jobs               map[string]Job
	Executors          map[string]Executor
	PipelineParameters map[string]Parameter

	ExecutorsRange          protocol.Range
	CommandsRange           protocol.Range
	JobsRange               protocol.Range
	PipelineParametersRange protocol.Range
	WorkflowRange           protocol.Range
	OrbsRange               protocol.Range
}

type RemoteOrbInfo struct {
	ID                 string
	FilePath           string
	Version            string
	LatestVersion      string
	LatestMinorVersion string
	LatestPatchVersion string
}

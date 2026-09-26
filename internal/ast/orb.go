package ast

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
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
	// IsPlaceholder is set for an orb declared as `{}`, a site that tooling
	// such as orb-tools/continue fills with an orb's source before the
	// config is run. What it will declare is not known.
	IsPlaceholder bool
}

type OrbURL struct {
	IsLocal bool
	// IsURL is set for an orb referenced by the URL of its source, such as
	// `https://example.com/orbs/go.yml`, which the compiler fetches from an
	// organization's allow-list. Name holds the URL, and there is no version.
	IsURL   bool
	Name    string
	Version string
}

type OrbURLDefinition struct {
	Namespace TextAndRange
	Name      TextAndRange
	Version   TextAndRange
}

// HasReference reports whether the orb is named with a `<< >>` template,
// such as `circleci/node@<< pipeline.parameters.version >>`, so that which
// orb it is is only known once the config is compiled.
func (orb OrbURL) HasReference() bool {
	return strings.Contains(orb.Name, "<<") || strings.Contains(orb.Version, "<<")
}

func (orb *OrbURL) GetOrbID() string {
	if orb.IsLocal || orb.IsURL {
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
	URI  uri.URI
	Name string

	Commands           map[string]Command
	Jobs               map[string]Job
	Executors          map[string]Executor
	PipelineParameters map[string]Parameter
	// Orbs and LocalOrbInfo are the orbs the orb declares for its own use.
	Orbs         map[string]Orb
	LocalOrbInfo map[string]*OrbInfo

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

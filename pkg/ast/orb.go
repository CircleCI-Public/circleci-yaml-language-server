package ast

import (
	"fmt"

	"go.lsp.dev/protocol"
)

type Orb struct {
	Url          OrbURL
	Name         string
	Range        protocol.Range
	NameRange    protocol.Range
	VersionRange protocol.Range
}

type OrbURL struct {
	IsLocal bool
	Name    string
	Version string
}

func (orb *OrbURL) GetOrbID() string {
	if orb.IsLocal {
		return orb.Name
	}
	return fmt.Sprintf("%s@%s", orb.Name, orb.Version)
}

type OrbInfo struct {
	IsLocal bool

	CreatedAt  string
	Commands   map[string]Command
	Jobs       map[string]Job
	Executors  map[string]Executor
	Parameters map[string]Parameter

	Description string
	Source      string
	RemoteInfo  RemoteOrbInfo

	OrbsRange       protocol.Range
	ExecutorsRange  protocol.Range
	CommandsRange   protocol.Range
	JobsRange       protocol.Range
	WorkflowRange   protocol.Range
	ParametersRange protocol.Range
}

type RemoteOrbInfo struct {
	ID                 string
	FilePath           string
	Version            string
	LatestVersion      string
	LatestMinorVersion string
	LatestPatchVersion string
}

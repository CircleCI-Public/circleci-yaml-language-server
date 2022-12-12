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
	Name    string
	Version string
}

func (orb *OrbURL) GetOrbID() string {
	return fmt.Sprintf("%s@%s", orb.Name, orb.Version)
}

type OrbInfo struct {
	CreatedAt   string
	Commands    map[string]Command
	Jobs        map[string]Job
	Executors   map[string]Executor
	Description string
	Source      string
	RemoteInfo  RemoteOrbInfo
	IsLocal     bool
}

type RemoteOrbInfo struct {
	ID                 string
	FilePath           string
	Version            string
	LatestVersion      string
	LatestMinorVersion string
	LatestPatchVersion string
}

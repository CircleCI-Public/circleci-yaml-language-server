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

type CachedOrb struct {
	ID                 string
	Version            string
	Source             string
	CreatedAt          string
	Commands           map[string]Command
	Jobs               map[string]Job
	Executors          map[string]Executor
	Description        string
	FilePath           string
	LatestVersion      string
	LatestMinorVersion string
	LatestPatchVersion string
}

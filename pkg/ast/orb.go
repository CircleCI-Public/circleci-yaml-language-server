package ast

import (
	"fmt"

	"go.lsp.dev/protocol"
)

type Orb struct {
	Range     protocol.Range
	NameRange protocol.Range
	Name      string
	Url       OrbURL
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

package ast

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

type Step interface {
	GetRange() protocol.Range

	GetName() string
}

type ParameterValue struct {
	Name       string
	Value      any
	ValueRange protocol.Range
	Range      protocol.Range
	Type       string
	Node       *sitter.Node
}
type NamedStep struct {
	Name            string
	Parameters      map[string]ParameterValue // Handle more values than just strings
	ParametersRange protocol.Range
	Range           protocol.Range
}

func (step NamedStep) GetRange() protocol.Range {
	return step.Range
}

func (step NamedStep) GetName() string {
	return step.Name
}

type Steps struct {
	Name            string
	Parameters      map[string]ParameterValue // Handle more values than just strings
	ParametersRange protocol.Range
	Range           protocol.Range
	Steps           []Step
}

func (step Steps) GetRange() protocol.Range {
	return step.Range
}

func (step Steps) GetName() string {
	return step.Name
}

type Run struct {
	protocol.Range
	Command          string
	CommandRange     protocol.Range
	RawCommand       string
	Name             string
	Shell            string
	Background       bool
	WorkingDirectory string
	NoOutputTimeout  string
	When             string
	WhenRange        protocol.Range
	Environment      map[string]string
}

func (step Run) GetRange() protocol.Range {
	return step.Range
}

func (step Run) GetName() string {
	return step.Name
}

type Checkout struct {
	protocol.Range
	Path string
}

func (step Checkout) GetRange() protocol.Range {
	return step.Range
}

func (step Checkout) GetName() string {
	return "checkout"
}

type SetupRemoteDocker struct {
	protocol.Range
	DockerLayerCaching bool
	Version            string
}

func (step SetupRemoteDocker) GetRange() protocol.Range {
	return step.Range
}

func (step SetupRemoteDocker) GetName() string {
	return "setup_remote_docker"
}

type SaveCache struct {
	protocol.Range
	Paths     []string
	Key       string
	CacheName string
	// When  // TODO
}

func (step SaveCache) GetRange() protocol.Range {
	return step.Range
}

func (step SaveCache) GetName() string {
	return "save_cache"
}

type RestoreCache struct {
	protocol.Range
	Key       string
	Keys      []string
	CacheName string
}

func (step RestoreCache) GetRange() protocol.Range {
	return step.Range
}

func (step RestoreCache) GetName() string {
	return "restore_cache"
}

type StoreArtifacts struct {
	protocol.Range
	Path        string
	Destination string
}

func (step StoreArtifacts) GetRange() protocol.Range {
	return step.Range
}

func (step StoreArtifacts) GetName() string {
	return "store_artifacts"
}

type StoreTestResults struct {
	protocol.Range
	Path string
}

func (step StoreTestResults) GetRange() protocol.Range {
	return step.Range
}

func (step StoreTestResults) GetName() string {
	return "store_test_results"
}

type PersistToWorkspace struct {
	protocol.Range
	Root  string
	Paths []string
}

func (step PersistToWorkspace) GetRange() protocol.Range {
	return step.Range
}

func (step PersistToWorkspace) GetName() string {
	return "persist_to_workspace"
}

type AttachWorkspace struct {
	protocol.Range
	At string
}

func (step AttachWorkspace) GetRange() protocol.Range {
	return step.Range
}

func (step AttachWorkspace) GetName() string {
	return "attach_workspace"
}

type AddSSHKey struct {
	protocol.Range
	Fingerprints []string
}

func (step AddSSHKey) GetRange() protocol.Range {
	return step.Range
}

func (step AddSSHKey) GetName() string {
	return "add_ssh_key"
}

package ast

import (
	"strings"

	"go.lsp.dev/protocol"
)

type Job struct {
	Range protocol.Range

	Name      string
	NameRange protocol.Range

	Shell            string
	WorkingDirectory string
	Parallelism      int
	ParallelismRange protocol.Range

	ResourceClass      string
	ResourceClassRange protocol.Range

	Steps      []Step
	StepsRange protocol.Range

	Description string

	Executor           string
	ExecutorParameters map[string]ParameterValue
	ExecutorRange      protocol.Range

	Parameters      map[string]Parameter
	ParametersRange protocol.Range

	Docker      DockerExecutor
	DockerRange protocol.Range

	CompletionItem *[]protocol.CompletionItem
}

func (job *Job) AddCompletionItem(label string, commitCharacters []string) {
	*job.CompletionItem = append(*job.CompletionItem, protocol.CompletionItem{
		Label:      label,
		Kind:       protocol.CompletionItemKindProperty,
		InsertText: label + strings.Join(commitCharacters, ""),
	})
}

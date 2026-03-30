package ast

import "go.lsp.dev/protocol"

type JobGroup struct {
	Range protocol.Range

	Name      string
	NameRange protocol.Range

	JobInvocations []JobInvocation
	JobsRange      protocol.Range
	JobsDAG        map[string][]string // Directed acyclic graph mapping each job to its requirements.
}

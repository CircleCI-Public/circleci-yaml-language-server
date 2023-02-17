package ast

import "go.lsp.dev/protocol"

type Workflow struct {
	protocol.Range
	Name      string
	NameRange protocol.Range
	JobRefs   []JobRef
	JobsDAG   map[string][]string // maps each job ref to its requirements, should be a directed acyclic graph

	HasTrigger bool
}

type JobRef struct {
	JobRefRange protocol.Range

	// JobName is the name of the job that will be executed,
	// it can be used to reference a job in the workflow
	JobName      string
	JobNameRange protocol.Range

	// StepName is the name of the job in the workflow,
	// not the job that will be executed
	StepName      string
	StepNameRange protocol.Range
	Requires      []TextAndRange
	Context       []TextAndRange
	Type          string
	TypeRange     protocol.Range
	Parameters    map[string]ParameterValue

	PreSteps      []Step
	PreStepsRange protocol.Range

	PostSteps      []Step
	PostStepsRange protocol.Range

	HasMatrix    bool
	MatrixParams map[string][]ParameterValue
}

package ast

import "go.lsp.dev/protocol"

type Workflow struct {
	protocol.Range
	Name      string
	NameRange protocol.Range
	JobsRange protocol.Range
	JobRefs   []JobRef
	JobsDAG   map[string][]string // maps each job ref to its requirements, should be a directed acyclic graph

	HasTrigger    bool
	Triggers      []WorkflowTrigger
	TriggersRange protocol.Range
}

type JobRef struct {
	JobRefRange protocol.Range

	// JobName is the name of the job that will be executed,
	// it can be used to reference a job in the workflow
	JobName      string
	JobNameRange protocol.Range

	// StepName is the name of the job in the workflow,
	// not the job that will be executed
	StepName          string
	StepNameRange     protocol.Range
	Requires          []Require
	Context           []TextAndRange
	Type              string
	TypeRange         protocol.Range
	Parameters        map[string]ParameterValue
	SerialGroup       string
	SerialGroupRange  protocol.Range
	OverrideWith      string
	OverrideWithRange protocol.Range

	PreSteps      []Step
	PreStepsRange protocol.Range

	PostSteps      []Step
	PostStepsRange protocol.Range

	HasMatrix    bool
	MatrixParams map[string][]ParameterValue
}

type Require struct {
	Name   string
	Status []string
	Range  protocol.Range
}

type WorkflowTrigger struct {
	Schedule ScheduleTrigger
	Range    protocol.Range
}

type ScheduleTrigger struct {
	Cron    string
	Filters WorkflowFilters
	Range   protocol.Range
}

type WorkflowFilters struct {
	Range    protocol.Range
	Branches BranchesFilter
}

type BranchesFilter struct {
	Range protocol.Range

	Only      []string
	OnlyRange protocol.Range

	Ignore      []string
	IgnoreRange protocol.Range
}

package ir

// Graph is the workflow graph of one compile. Its jobs are held by Ref, so
// the graph's own ref covers them.
type Graph struct {
	Version   string     `json:"version"` // "graph/1"
	Setup     bool       `json:"setup,omitempty"`
	Workflows []Workflow `json:"workflows"` // sorted by name
}

type Workflow struct {
	Name          string `json:"name"`
	MaxAutoReruns *int   `json:"max_auto_reruns,omitempty"`
	Nodes         []Node `json:"nodes"` // sorted by name
	Loc           Loc    `json:"loc"`
}

type Node struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"`          // build, approval, no-op, release, lock, unlock
	Job            Ref               `json:"job,omitempty"` // build only
	Upstream       []Edge            `json:"upstream,omitempty"`
	Contexts       []string          `json:"contexts,omitempty"`
	SerialGroupIDs []string          `json:"serial_group_ids,omitempty"`
	Key            string            `json:"key,omitempty"` // lock, unlock
	Release        *Release          `json:"release,omitempty"`
	Matrix         map[string]string `json:"matrix,omitempty"`
	Origin         []Frame           `json:"origin"`
}

type Edge struct {
	Node     string   `json:"node"`
	IfStatus []string `json:"if_status"` // sorted, never empty; see Statuses
}

type Release struct {
	PlanName   string `json:"plan_name"`
	Validation any    `json:"validation,omitempty"`
}

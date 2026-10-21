package ir

// Job is one fully expanded job. Optional scalars whose zero value means
// something are pointers; everything else is omitted when empty, so a missing
// key is the only spelling of unset.
type Job struct {
	Version          string   `json:"version"` // "job/1"
	Executor         Executor `json:"executor"`
	Shell            string   `json:"shell,omitempty"`
	WorkingDirectory string   `json:"working_directory,omitempty"`
	Parallelism      *int     `json:"parallelism,omitempty"`
	Environment      []Env    `json:"environment,omitempty"`
	Retention        string   `json:"retention,omitempty"`
	Steps            []Step   `json:"steps"`
	Loc              Loc      `json:"loc"`
}

type Executor struct {
	Kind          string  `json:"kind"` // docker, machine, macos
	ResourceClass string  `json:"resource_class,omitempty"`
	Images        []Image `json:"images,omitempty"`
	MachineImage  string  `json:"machine_image,omitempty"`
	DLC           *bool   `json:"dlc,omitempty"`
	Xcode         string  `json:"xcode,omitempty"`
	Origin        []Frame `json:"origin,omitempty"`
}

type Image struct {
	Name        string     `json:"name"`
	Entrypoint  []string   `json:"entrypoint,omitempty"`
	Command     []string   `json:"command,omitempty"`
	User        string     `json:"user,omitempty"`
	Environment []Env      `json:"environment,omitempty"`
	Auth        *ImageAuth `json:"auth,omitempty"`
}

type ImageAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Step struct {
	Kind   string         `json:"kind"`
	Name   string         `json:"name,omitempty"`
	When   string         `json:"when,omitempty"`
	Args   map[string]any `json:"args,omitempty"`
	Origin []Frame        `json:"origin"`
}

type Env struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

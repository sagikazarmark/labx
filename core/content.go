package core

import (
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
)

type ContentManifest struct {
	// Metadata preserves additional upstream front-matter fields without interpreting them.
	Metadata     map[string]any        `yaml:",inline"                json:"-"`
	Difficulties []string              `yaml:"difficulties,omitempty" json:"difficulties,omitempty"`
	Kind         content.ContentKind   `yaml:"kind"                   json:"kind"`
	Title        string                `yaml:"title"                  json:"title"`
	Description  string                `yaml:"description"            json:"description"`
	Categories   []string              `yaml:"categories,omitempty"   json:"categories,omitempty"`
	Tags         []string              `yaml:"tagz,omitempty"         json:"tagz,omitempty"`
	CreatedAt    string                `yaml:"createdAt"              json:"createdAt"`
	UpdatedAt    string                `yaml:"updatedAt,omitempty"    json:"updatedAt,omitempty"`
	Cover        string                `yaml:"cover,omitempty"        json:"cover,omitempty"`
	Playground   ContentPlaygroundSpec `yaml:"playground,omitempty"   json:"playground,omitzero"`
	Tasks        map[string]Task       `yaml:"tasks,omitempty"        json:"tasks,omitzero"`
	// Vars preserves Labs runtime declarations: strings or shell/machine objects.
	Vars       map[string]any `yaml:"vars,omitempty"       json:"vars,omitempty"`
	ShellGym   *ShellGym      `yaml:"shellgym,omitempty"   json:"shellgym,omitempty"`
	Repetition *[]int         `yaml:"repetition,omitempty" json:"repetition,omitempty"`

	// Challenge specific fields
	Difficulty string `yaml:"difficulty,omitempty" json:"difficulty,omitempty"`

	// Course specific fields
	Slug string `yaml:"slug,omitempty" json:"slug,omitempty"`
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Content embedding
	Challenges  map[string]struct{} `yaml:"challenges,omitempty"  json:"challenges,omitempty"`
	Tutorials   map[string]struct{} `yaml:"tutorials,omitempty"   json:"tutorials,omitempty"`
	Playgrounds map[string]struct{} `yaml:"playgrounds,omitempty" json:"playgrounds,omitempty"`
	SkillPaths  map[string]struct{} `yaml:"skill-paths,omitempty" json:"skill-paths,omitempty"`

	// Training specific fields
	WorkingTitle string `yaml:"workingTitle,omitempty" json:"workingTitle,omitempty"`
}

type ContentPlaygroundSpec struct {
	Metadata       map[string]any             `yaml:",inline"                  json:"-"`
	InitTasks      map[string]InitTask        `yaml:"initTasks,omitempty"      json:"initTasks,omitempty"`
	InitConditions api.InitConditions         `yaml:"initConditions,omitempty" json:"initConditions,omitempty"`
	PortForwards   []api.PortForward          `yaml:"portForwards,omitempty"   json:"portForwards,omitempty"`
	StartupFiles   []StartupFile              `yaml:"startupFiles,omitempty"   json:"startupFiles,omitempty"`
	RegistryAuth   string                     `yaml:"registryAuth,omitempty"   json:"registryAuth,omitempty"`
	Name           string                     `yaml:"name,omitempty"           json:"name,omitempty"`
	Networks       []api.PlaygroundNetwork    `yaml:"networks,omitempty"       json:"networks,omitempty"`
	Machines       []ContentPlaygroundMachine `yaml:"machines,omitempty"       json:"machines,omitempty"`
	Tabs           []api.PlaygroundTab        `yaml:"tabs,omitempty"           json:"tabs,omitempty"`
}

type ContentPlaygroundMachine struct {
	Metadata     map[string]any        `yaml:",inline"                json:"-"`
	Name         string                `yaml:"name"                   json:"name"`
	Users        []MachineUser         `yaml:"users,omitempty"        json:"users,omitempty"`
	Backend      api.MachineBackend    `yaml:"backend,omitempty"      json:"backend,omitempty"`
	Kernel       *api.MachineKernel    `yaml:"kernel,omitempty"       json:"kernel,omitempty"`
	Drives       []api.MachineDrive    `yaml:"drives,omitempty"       json:"drives,omitempty"`
	Network      *api.MachineNetwork   `yaml:"network,omitzero"       json:"network,omitzero"`
	Resources    *api.MachineResources `yaml:"resources,omitzero"     json:"resources,omitzero"`
	StartupFiles []StartupFile         `yaml:"startupFiles,omitempty" json:"startupFiles,omitempty"`
	NoSSH        bool                  `yaml:"noSSH,omitzero"         json:"noSSH,omitzero"`
}

type Task struct {
	Metadata       map[string]any `yaml:",inline"           json:"-"`
	Machine        string         `yaml:"machine,omitempty" json:"machine,omitempty"`
	Init           bool           `yaml:"init,omitempty"    json:"init,omitempty"`
	Helper         bool           `yaml:"helper,omitempty"  json:"helper,omitempty"`
	User           string         `yaml:"user,omitempty"    json:"user,omitempty"`
	TimeoutSeconds int            `yaml:"timeout_seconds"   json:"timeout_seconds"`
	Needs          []string       `yaml:"needs,omitempty"   json:"needs,omitempty"`
	Env            []string       `yaml:"env,omitempty"     json:"env,omitempty"`
	Run            string         `yaml:"run"               json:"run"`
	HintCheck      string         `yaml:"hintcheck"         json:"hintcheck"`
	FailCheck      string         `yaml:"failcheck"         json:"failcheck"`
}

// Pointers distinguish omitted Shell Gym settings from explicitly supplied zero
// values. Labs owns defaults and validation.
type ShellGym struct {
	Metadata map[string]any `yaml:",inline"           json:"-"`
	Version  *string        `yaml:"version,omitempty" json:"version,omitempty"`
	Path     *string        `yaml:"path,omitempty"    json:"path,omitempty"`
	Port     *int           `yaml:"port,omitempty"    json:"port,omitempty"`
}

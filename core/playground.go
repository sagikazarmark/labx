package core

import "github.com/iximiuz/labctl/api"

// Local YAML models preserve fields unknown to labx while upstream API types
// remain the boundary for data fetched from labctl.
type PlaygroundManifest struct {
	Metadata    map[string]any `yaml:",inline"               json:"-"`
	Kind        string         `yaml:"kind"                  json:"kind"`
	Name        string         `yaml:"name"                  json:"name"`
	Base        string         `yaml:"base,omitempty"        json:"base,omitempty"`
	Title       string         `yaml:"title,omitempty"       json:"title,omitempty"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	Cover       string         `yaml:"cover,omitempty"       json:"cover,omitempty"`
	Categories  []string       `yaml:"categories,omitempty"  json:"categories,omitempty"`
	Markdown    string         `yaml:"markdown,omitempty"    json:"markdown,omitempty"`
	Playground  PlaygroundSpec `yaml:"playground"            json:"playground"`
}

type PlaygroundSpec struct {
	Metadata       map[string]any              `yaml:",inline"                  json:"-"`
	Networks       []api.PlaygroundNetwork     `yaml:"networks,omitempty"       json:"networks,omitempty"`
	Machines       []ContentPlaygroundMachine  `yaml:"machines,omitempty"       json:"machines,omitempty"`
	StartupFiles   []StartupFile               `yaml:"startupFiles,omitempty"   json:"startupFiles,omitempty"`
	Tabs           []api.PlaygroundTab         `yaml:"tabs,omitempty"           json:"tabs,omitempty"`
	InitTasks      map[string]InitTask         `yaml:"initTasks,omitempty"      json:"initTasks,omitempty"`
	InitConditions api.InitConditions          `yaml:"initConditions,omitempty" json:"initConditions,omitempty"`
	RegistryAuth   string                      `yaml:"registryAuth,omitempty"   json:"registryAuth,omitempty"`
	PortForwards   []api.PortForward           `yaml:"portForwards,omitempty"   json:"portForwards,omitempty"`
	AccessControl  api.PlaygroundAccessControl `yaml:"accessControl,omitempty"  json:"accessControl,omitempty"`
}

type InitTask struct {
	Metadata       map[string]any      `yaml:",inline"                   json:"-"`
	Name           string              `yaml:"name,omitempty"            json:"name,omitempty"`
	Machine        string              `yaml:"machine,omitempty"         json:"machine,omitempty"`
	Init           bool                `yaml:"init,omitempty"            json:"init,omitempty"`
	User           string              `yaml:"user,omitempty"            json:"user,omitempty"`
	TimeoutSeconds int                 `yaml:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`
	Needs          []string            `yaml:"needs,omitempty"           json:"needs,omitempty"`
	Run            string              `yaml:"run"                       json:"run"`
	Conditions     []api.InitCondition `yaml:"conditions,omitempty"      json:"conditions,omitempty"`
}

type StartupFile struct {
	Metadata map[string]any `yaml:",inline"            json:"-"`
	Path     string         `yaml:"path"               json:"path"`
	Content  string         `yaml:"content,omitempty"  json:"content,omitempty"`
	Source   string         `yaml:"source,omitempty"   json:"source,omitempty"`
	Mode     string         `yaml:"mode,omitempty"     json:"mode,omitempty"`
	Owner    string         `yaml:"owner,omitempty"    json:"owner,omitempty"`
	Append   bool           `yaml:"append,omitempty"   json:"append,omitempty"`
	Extract  bool           `yaml:"extract,omitempty"  json:"extract,omitempty"`
	Machines []string       `yaml:"machines,omitempty" json:"machines,omitempty"`
}

type MachineUser struct {
	Metadata map[string]any `yaml:",inline"           json:"-"`
	Name     string         `yaml:"name"              json:"name"`
	Default  bool           `yaml:"default,omitempty" json:"default,omitempty"`
	Welcome  string         `yaml:"welcome,omitempty" json:"welcome,omitempty"`
}

func StartupFilesFromAPI(files []api.StartupFile) []StartupFile {
	var result []StartupFile
	for _, file := range files {
		result = append(result, StartupFile{
			Path: file.Path, Content: file.Content, Source: file.Source,
			Mode: file.Mode, Owner: file.Owner, Append: file.Append,
			Extract: file.Extract, Machines: file.Machines,
		})
	}
	return result
}

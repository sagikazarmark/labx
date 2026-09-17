package extended

import (
	"fmt"
	"slices"

	"github.com/iximiuz/labctl/api"
	"github.com/samber/lo"

	"github.com/sagikazarmark/labx/core"
)

type PlaygroundManifest struct {
	Metadata    map[string]any     `yaml:",inline"     json:"-"`
	Kind        string             `yaml:"kind"        json:"kind"`
	Name        string             `yaml:"name"        json:"name"`
	Base        string             `yaml:"base"        json:"base"`
	Title       string             `yaml:"title"       json:"title"`
	Description string             `yaml:"description" json:"description"`
	Channels    map[string]Channel `yaml:"channels"    json:"channels"`
	Cover       string             `yaml:"cover"       json:"cover"`
	Categories  []string           `yaml:"categories"  json:"categories"`
	Markdown    string             `yaml:"markdown"    json:"markdown"`
	Playground  PlaygroundSpec     `yaml:"playground"  json:"playground"`
}

func (m PlaygroundManifest) Convert() core.PlaygroundManifest {
	return core.PlaygroundManifest{
		Metadata:    m.Metadata,
		Kind:        m.Kind,
		Name:        m.Name,
		Base:        m.Base,
		Title:       m.Title,
		Description: m.Description,
		Cover:       m.Cover,
		Categories:  m.Categories,
		Markdown:    m.Markdown,
		Playground:  m.Playground.Convert(),
	}
}

type PlaygroundSpec struct {
	Metadata       map[string]any          `yaml:",inline"                json:"-"`
	StartupFiles   MachineStartupFiles     `yaml:"startupFiles,omitempty" json:"startupFiles,omitempty"`
	PortForwards   []api.PortForward       `yaml:"portForwards,omitempty" json:"portForwards,omitempty"`
	Welcome        string                  `yaml:"welcome"                json:"welcome"`
	Networks       []api.PlaygroundNetwork `yaml:"networks"               json:"networks"`
	Machines       PlaygroundMachines      `yaml:"machines"               json:"machines"`
	Tabs           []api.PlaygroundTab     `yaml:"tabs"                   json:"tabs"`
	InitTasks      InitTasks               `yaml:"initTasks"              json:"initTasks"`
	InitConditions api.InitConditions      `yaml:"initConditions"         json:"initConditions"`
	RegistryAuth   string                  `yaml:"registryAuth,omitempty" json:"registryAuth,omitempty"`

	AccessControl api.PlaygroundAccessControl `yaml:"accessControl" json:"accessControl"`

	BaseName string `yaml:"-" json:"-"`
}

func (s PlaygroundSpec) Convert() core.PlaygroundSpec {
	return core.PlaygroundSpec{
		Metadata:       s.Metadata,
		StartupFiles:   s.StartupFiles.Convert(),
		PortForwards:   s.PortForwards,
		Networks:       s.Networks,
		Machines:       s.convertMachines(),
		Tabs:           s.Tabs,
		InitTasks:      s.InitTasks.Convert(),
		InitConditions: s.InitConditions,
		RegistryAuth:   s.RegistryAuth,
		AccessControl:  s.AccessControl,
	}
}

func (s PlaygroundSpec) convertMachines() []core.ContentPlaygroundMachine {
	if s.BaseName == "flexbox" {
		return s.Machines.Convert()
	}

	// Apply welcome message to default users if specified
	machines := s.Machines.Convert()
	if s.Welcome != "" {
		for i, machine := range machines {
			for j, user := range machine.Users {
				if user.Default && (user.Welcome == "" || user.Welcome == "-") {
					machines[i].Users[j].Welcome = s.Welcome
				}
			}
		}
	}

	return machines
}

type PlaygroundMachines []PlaygroundMachine

func (m PlaygroundMachines) Convert() []core.ContentPlaygroundMachine {
	return lo.Map(m, func(machine PlaygroundMachine, _ int) core.ContentPlaygroundMachine {
		return machine.Convert()
	})
}

type PlaygroundMachine struct {
	Metadata     map[string]any        `yaml:",inline"            json:"-"`
	Name         string                `yaml:"name"               json:"name"`
	Hostname     string                `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	IDEPath      string                `yaml:"idePath,omitempty"  json:"idePath,omitempty"`
	Users        MachineUsers          `yaml:"users"              json:"users"`
	Backend      api.MachineBackend    `yaml:"backend,omitempty"  json:"backend,omitempty"`
	Kernel       *api.MachineKernel    `yaml:"kernel,omitempty"   json:"kernel,omitempty"`
	Drives       []api.MachineDrive    `yaml:"drives"             json:"drives"`
	Network      *api.MachineNetwork   `yaml:"network,omitzero"   json:"network,omitzero"`
	Resources    *api.MachineResources `yaml:"resources,omitzero" json:"resources,omitzero"`
	StartupFiles MachineStartupFiles   `yaml:"startupFiles"       json:"startupFiles"`
	NoSSH        bool                  `yaml:"noSSH,omitzero"     json:"noSSH,omitzero"`
}

func (m PlaygroundMachine) Convert() core.ContentPlaygroundMachine {
	var playgroundStartupFiles []core.StartupFile

	if m.Hostname != "" {
		hostname := core.StartupFile{
			Path:    "/etc/hostname",
			Content: m.Hostname,
			Mode:    "755",
			Owner:   "root:root",
		}

		hosts := core.StartupFile{
			Path:    "/etc/hosts",
			Content: fmt.Sprintf("127.0.0.1       %s %s.local\n", m.Hostname, m.Hostname),
			Append:  true,
		}

		playgroundStartupFiles = append(playgroundStartupFiles, hostname, hosts)
	}

	if m.IDEPath != "" {
		codeServerEnv := core.StartupFile{
			Path:    "/etc/default/code-server",
			Content: fmt.Sprintf("CODE_SERVER_PATH=%s\n", m.IDEPath),
			Owner:   "root:root",
			Mode:    "644",
		}

		playgroundStartupFiles = append(playgroundStartupFiles, codeServerEnv)
	}

	return core.ContentPlaygroundMachine{
		Metadata:     m.Metadata,
		Name:         m.Name,
		Users:        m.Users.Convert(),
		Backend:      m.Backend,
		Kernel:       m.Kernel,
		Drives:       m.Drives,
		Network:      m.Network,
		Resources:    m.Resources,
		StartupFiles: append(playgroundStartupFiles, m.StartupFiles.Convert()...),
		NoSSH:        m.NoSSH,
	}
}

type MachineUsers []MachineUser

func (u MachineUsers) Convert() []core.MachineUser {
	return lo.Map(u, func(user MachineUser, _ int) core.MachineUser {
		return user.Convert()
	})
}

type MachineUser struct {
	Metadata    map[string]any `yaml:",inline"               json:"-"`
	Name        string         `yaml:"name"                  json:"name"`
	Default     bool           `yaml:"default,omitempty"     json:"default,omitempty"`
	Welcome     string         `yaml:"welcome,omitempty"     json:"welcome,omitempty"`
	WelcomeFile string         `yaml:"welcomeFile,omitempty" json:"welcomeFile,omitempty"`
}

func (u MachineUser) Convert() core.MachineUser {
	return core.MachineUser{
		Metadata: u.Metadata,
		Name:     u.Name,
		Default:  u.Default,
		Welcome:  u.Welcome,
	}
}

type MachineStartupFiles []MachineStartupFile

func (m MachineStartupFiles) Convert() []core.StartupFile {
	return lo.Map(m, func(file MachineStartupFile, _ int) core.StartupFile {
		return file.Convert()
	})
}

type MachineStartupFile struct {
	Metadata map[string]any `yaml:",inline"            json:"-"`
	Source   string         `yaml:"source,omitempty"   json:"source,omitempty"`
	Extract  bool           `yaml:"extract,omitempty"  json:"extract,omitempty"`
	Machines []string       `yaml:"machines,omitempty" json:"machines,omitempty"`
	Path     string         `yaml:"path"               json:"path"`
	FromFile string         `yaml:"fromFile,omitempty" json:"fromFile,omitempty"`
	Content  string         `yaml:"content,omitempty"  json:"content,omitempty"`
	Mode     string         `yaml:"mode,omitempty"     json:"mode,omitempty"`
	Owner    string         `yaml:"owner,omitempty"    json:"owner,omitempty"`
	Append   bool           `yaml:"append,omitempty"   json:"append,omitempty"`
}

func (f MachineStartupFile) Convert() core.StartupFile {
	return core.StartupFile{
		Metadata: f.Metadata,
		Source:   f.Source,
		Extract:  f.Extract,
		Machines: slices.Clone(f.Machines),
		Path:     f.Path,
		Content:  f.Content,
		Mode:     f.Mode,
		Owner:    f.Owner,
		Append:   f.Append,
	}
}

type InitTasks map[string]InitTask

func (t InitTasks) Convert() map[string]core.InitTask {
	return t.convert(nil)
}

// Parent tasks can satisfy dependencies without being emitted again as overrides.
func (t InitTasks) convert(parent map[string]api.InitTask) map[string]core.InitTask {
	initTasks := map[string]core.InitTask{}

	for name, initTask := range t {
		for _, machine := range taskTargets(initTask.Machine) {
			for _, user := range taskTargets(initTask.User) {
				newInitTask := initTask.ConvertCurrent(name, machine, user)

				// Dependency check and resolution
				for i, need := range newInitTask.Needs {
					// Dependency found with this name; need to check dependency resolution rules
					if dep, ok := t[need]; ok {
						// Theoretically, this is now supported
						// TODO: remove once confirmed
						// ~~Dependency must always run on the same machine~~
						// if !slices.Contains(dep.Machine, machine) {
						// 	panic("invalid dependency: machine")
						// }

						// Dependency must have the same user in the list when running as multiple users
						if len(dep.User) > 1 && !slices.Contains(dep.User, user) {
							panic("invalid dependency: user")
						}

						newInitTask.Needs[i] = dep.currentName(need, machine, user)

						continue
					}

					// Dependency not found with this name so try a few other options
					// TODO: is this necessary?

					// Machine name AND user manually added
					if dep, ok := t[taskName(need, machine, user)]; ok {
						newInitTask.Needs[i] = dep.Name

						continue
					}

					// Machine name manually added
					if dep, ok := t[taskName(need, machine)]; ok {
						newInitTask.Needs[i] = dep.Name

						continue
					}

					if _, ok := parent[need]; ok {
						continue
					}
					// dependency not found anywhere
					panic("unknown dependency:" + need)
				}

				initTasks[newInitTask.Name] = newInitTask
			}
		}
	}

	return initTasks
}

type InitTask struct {
	Metadata       map[string]any      `yaml:",inline"              json:"-"`
	Name           string              `yaml:"name"                 json:"name"`
	Machine        StringList          `yaml:"machine,omitempty"    json:"machine,omitempty"`
	Init           bool                `yaml:"init"                 json:"init"`
	User           StringList          `yaml:"user"                 json:"user"`
	TimeoutSeconds int                 `yaml:"timeout_seconds"      json:"timeout_seconds"`
	Needs          []string            `yaml:"needs,omitempty"      json:"needs,omitempty"`
	Run            string              `yaml:"run"                  json:"run"`
	Conditions     []api.InitCondition `yaml:"conditions,omitempty" json:"conditions,omitempty"`
}

func (t InitTask) Convert() core.InitTask {
	return core.InitTask{
		Metadata:       t.Metadata,
		Name:           t.Name,
		Init:           t.Init,
		TimeoutSeconds: t.TimeoutSeconds,
		Needs:          slices.Clone(t.Needs),
		Run:            t.Run,
		Conditions:     slices.Clone(t.Conditions),
	}
}

func (t InitTask) ConvertCurrent(name string, machine string, user string) core.InitTask {
	initTask := t.Convert()
	initTask.Machine = machine
	initTask.User = user
	initTask.Name = t.currentName(name, machine, user)

	return initTask
}

func (t InitTask) currentName(name string, machine string, user string) string {
	var taskNameSegments []string

	if len(t.Machine) > 1 {
		taskNameSegments = append(taskNameSegments, machine)
	}

	if len(t.User) > 1 {
		taskNameSegments = append(taskNameSegments, user)
	}

	if t.Name != "" {
		name = t.Name
	}

	return taskName(name, taskNameSegments...)
}

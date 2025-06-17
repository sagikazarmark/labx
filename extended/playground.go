package extended

import (
	"fmt"
	"slices"

	"github.com/iximiuz/labctl/api"
	"github.com/samber/lo"
)

type PlaygroundManifest struct {
	Kind        string             `yaml:"kind" json:"kind"`
	Name        string             `yaml:"name" json:"name"`
	Base        string             `yaml:"base" json:"base"`
	Title       string             `yaml:"title" json:"title"`
	Description string             `yaml:"description" json:"description"`
	Channels    map[string]Channel `yaml:"channels" json:"channels"`
	Cover       string             `yaml:"cover" json:"cover"`
	Categories  []string           `yaml:"categories" json:"categories"`
	Markdown    string             `yaml:"markdown" json:"markdown"`
	Playground  PlaygroundSpec     `yaml:"playground" json:"playground"`
}

func (m PlaygroundManifest) Convert() api.PlaygroundManifest {
	return api.PlaygroundManifest{
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
	Welcome        string                  `yaml:"welcome" json:"welcome"`
	Networks       []api.PlaygroundNetwork `yaml:"networks" json:"networks"`
	Machines       PlaygroundMachines      `yaml:"machines" json:"machines"`
	Tabs           []api.PlaygroundTab     `yaml:"tabs" json:"tabs"`
	InitTasks      InitTasks               `yaml:"initTasks" json:"initTasks"`
	InitConditions api.InitConditions      `yaml:"initConditions" json:"initConditions"`
	RegistryAuth   string                  `yaml:"registryAuth,omitempty" json:"registryAuth,omitempty"`

	AccessControl api.PlaygroundAccessControl `yaml:"accessControl" json:"accessControl"`

	BaseName string             `yaml:"-" json:"-"`
	Base     api.PlaygroundSpec `yaml:"-" json:"-"`
}

func (s PlaygroundSpec) Convert() api.PlaygroundSpec {
	return api.PlaygroundSpec{
		Networks:       s.Networks,
		Machines:       s.convertMachines(),
		Tabs:           s.Tabs,
		InitTasks:      s.InitTasks.Convert(),
		InitConditions: s.InitConditions,
		RegistryAuth:   s.RegistryAuth,
		AccessControl:  s.AccessControl,
	}
}

func (s PlaygroundSpec) convertMachines() []api.PlaygroundMachine {
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

func (m PlaygroundMachines) Convert() []api.PlaygroundMachine {
	return lo.Map(m, func(machine PlaygroundMachine, _ int) api.PlaygroundMachine {
		return machine.Convert()
	})
}

type PlaygroundMachine struct {
	Name         string               `yaml:"name" json:"name"`
	Hostname     string               `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	IDEPath      string               `yaml:"idePath,omitempty" json:"idePath,omitempty"`
	Users        MachineUsers         `yaml:"users" json:"users"`
	Kernel       string               `yaml:"kernel,omitempty" json:"kernel,omitempty"`
	Drives       []api.MachineDrive   `yaml:"drives" json:"drives"`
	Network      api.MachineNetwork   `yaml:"network" json:"network"`
	Resources    api.MachineResources `yaml:"resources" json:"resources"`
	StartupFiles MachineStartupFiles  `yaml:"startupFiles" json:"startupFiles"`
}

func (m PlaygroundMachine) Convert() api.PlaygroundMachine {
	var playgroundStartupFiles []api.MachineStartupFile

	if m.Hostname != "" {
		hostname := api.MachineStartupFile{
			Path:    "/etc/hostname",
			Content: m.Hostname,
			Mode:    "755",
			Owner:   "root:root",
		}

		hosts := api.MachineStartupFile{
			Path:    "/etc/hosts",
			Content: fmt.Sprintf("127.0.0.1       %s %s.local\n", m.Hostname, m.Hostname),
			Append:  true,
		}

		playgroundStartupFiles = append(playgroundStartupFiles, hostname, hosts)
	}

	if m.IDEPath != "" {
		codeServerEnv := api.MachineStartupFile{
			Path:    "/etc/default/code-server",
			Content: fmt.Sprintf("CODE_SERVER_PATH=%s\n", m.IDEPath),
			Owner:   "root:root",
			Mode:    "644",
		}

		playgroundStartupFiles = append(playgroundStartupFiles, codeServerEnv)
	}

	return api.PlaygroundMachine{
		Name:         m.Name,
		Users:        m.Users.Convert(),
		Kernel:       m.Kernel,
		Drives:       m.Drives,
		Network:      m.Network,
		Resources:    m.Resources,
		StartupFiles: append(playgroundStartupFiles, m.StartupFiles.Convert()...),
	}
}

type MachineUsers []MachineUser

func (u MachineUsers) Convert() []api.MachineUser {
	return lo.Map(u, func(user MachineUser, _ int) api.MachineUser {
		return user.Convert()
	})
}

type MachineUser struct {
	Name        string `yaml:"name" json:"name"`
	Default     bool   `yaml:"default,omitempty" json:"default,omitempty"`
	Welcome     string `yaml:"welcome,omitempty" json:"welcome,omitempty"`
	WelcomeFile string `yaml:"welcomeFile,omitempty" json:"welcomeFile,omitempty"`
}

func (u MachineUser) Convert() api.MachineUser {
	return api.MachineUser{
		Name:    u.Name,
		Default: u.Default,
		Welcome: u.Welcome,
	}
}

type MachineStartupFiles []MachineStartupFile

func (m MachineStartupFiles) Convert() []api.MachineStartupFile {
	return lo.Map(m, func(file MachineStartupFile, _ int) api.MachineStartupFile {
		return file.Convert()
	})
}

type MachineStartupFile struct {
	Path     string `yaml:"path" json:"path"`
	FromFile string `yaml:"fromFile,omitempty" json:"fromFile,omitempty"`
	Content  string `yaml:"content,omitempty" json:"content,omitempty"`
	Mode     string `yaml:"mode,omitempty" json:"mode,omitempty"`
	Owner    string `yaml:"owner,omitempty" json:"owner,omitempty"`
	Append   bool   `yaml:"append,omitempty" json:"append,omitempty"`
}

func (f MachineStartupFile) Convert() api.MachineStartupFile {
	return api.MachineStartupFile{
		Path:    f.Path,
		Content: f.Content,
		Mode:    f.Mode,
		Owner:   f.Owner,
		Append:  f.Append,
	}
}

type InitTasks map[string]InitTask

func (t InitTasks) Convert() map[string]api.InitTask {
	initTasks := map[string]api.InitTask{}

	for name, initTask := range t {
		for _, machine := range initTask.Machine {
			for _, user := range initTask.User {
				newInitTask := initTask.ConvertCurrent(name, machine, user)

				// Dependency check and resolution
				for i, need := range newInitTask.Needs {
					// Dependency found with this name; need to check dependency resolution rules
					if dep, ok := t[need]; ok {
						// Dependency must always run on the same machine
						if !slices.Contains(dep.Machine, machine) {
							panic("invalid dependency: machine")
						}

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
	Name           string              `yaml:"name" json:"name"`
	Machine        StringList          `yaml:"machine,omitempty" json:"machine,omitempty"`
	Init           bool                `yaml:"init" json:"init"`
	User           StringList          `yaml:"user" json:"user"`
	TimeoutSeconds int                 `yaml:"timeout_seconds" json:"timeout_seconds"`
	Needs          []string            `yaml:"needs,omitempty" json:"needs,omitempty"`
	Run            string              `yaml:"run" json:"run"`
	Conditions     []api.InitCondition `yaml:"conditions,omitempty" json:"conditions,omitempty"`
}

func (t InitTask) Convert() api.InitTask {
	return api.InitTask{
		Name:           t.Name,
		Init:           t.Init,
		TimeoutSeconds: t.TimeoutSeconds,
		Needs:          slices.Clone(t.Needs),
		Run:            t.Run,
		Conditions:     slices.Clone(t.Conditions),
	}
}

func (t InitTask) ConvertCurrent(name string, machine string, user string) api.InitTask {
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

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
	Welcome        string              `yaml:"welcome" json:"welcome"`
	Machines       PlaygroundMachines  `yaml:"machines" json:"machines"`
	Tabs           []api.PlaygroundTab `yaml:"tabs" json:"tabs"`
	InitTasks      InitTasks           `yaml:"initTasks" json:"initTasks"`
	InitConditions api.InitConditions  `yaml:"initConditions" json:"initConditions"`
	RegistryAuth   string              `yaml:"registryAuth,omitempty" json:"registryAuth,omitempty"`

	AccessControl api.PlaygroundAccessControl `yaml:"accessControl" json:"accessControl"`

	Base api.PlaygroundSpec `yaml:"-" json:"-"`
}

func (s PlaygroundSpec) Convert() api.PlaygroundSpec {
	return api.PlaygroundSpec{
		Machines:       s.convertMachines(),
		Tabs:           s.Tabs,
		InitTasks:      s.InitTasks.Convert(),
		InitConditions: s.InitConditions,
		RegistryAuth:   s.RegistryAuth,
		AccessControl:  s.AccessControl,
	}
}

func (s PlaygroundSpec) convertMachines() []api.PlaygroundMachine {
	parentMachines := lo.SliceToMap(s.Base.Machines, func(machine api.PlaygroundMachine) (string, api.PlaygroundMachine) {
		return machine.Name, machine
	})

	// Make sure to include startup files, users and resources from parent playground
	return lo.Map(s.Machines.Convert(), func(machine api.PlaygroundMachine, _ int) api.PlaygroundMachine {
		parentMachine := parentMachines[machine.Name]

		if len(machine.Users) == 0 {
			machine.Users = slices.Clone(parentMachine.Users)
		}

		if s.Welcome != "" {
			for i, user := range machine.Users {
				if !user.Default {
					continue
				}

				if user.Welcome == "" || user.Welcome == "-" {
					machine.Users[i].Welcome = s.Welcome
				}
			}
		}

		if machine.Resources.CPUCount == 0 {
			machine.Resources.CPUCount = parentMachine.Resources.CPUCount
		}

		if machine.Resources.RAMSize == "" {
			machine.Resources.RAMSize = parentMachine.Resources.RAMSize
		}

		return machine
	})
}

type PlaygroundMachines []PlaygroundMachine

func (m PlaygroundMachines) Convert() []api.PlaygroundMachine {
	return lo.Map(m, func(machine PlaygroundMachine, _ int) api.PlaygroundMachine {
		return machine.Convert()
	})
}

type PlaygroundMachine struct {
	Name         string                   `yaml:"name" json:"name"`
	Hostname     string                   `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	IDEPath      string                   `yaml:"idePath,omitempty" json:"idePath,omitempty"`
	Users        []api.MachineUser        `yaml:"users" json:"users"`
	Resources    api.MachineResources     `yaml:"resources" json:"resources"`
	StartupFiles []api.MachineStartupFile `yaml:"startupFiles" json:"startupFiles"`
}

const codeServerUnit = `[Unit]
Description=code-server

[Service]
Type=exec
Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin:/usr/local/go/bin:/home/laborant/go/bin" "HOME=/home/laborant"
User=laborant
ExecStart=/usr/bin/code-server --bind-addr=127.0.0.1:50062 --auth none --disable-telemetry --disable-update-check --disable-workspace-trust --disable-getting-started-override --app-name="iximiuz Labs" $CODE_SERVER_PATH
Restart=on-failure
Environment=CODE_SERVER_PATH=/home/laborant
Environment=CODE_SERVER_PATH=%s
`

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
		unit := api.MachineStartupFile{
			Path:    "/usr/lib/systemd/system/code-server.service",
			Content: fmt.Sprintf(codeServerUnit, m.IDEPath),
			Owner:   "root:root",
			Mode:    "644",
		}

		playgroundStartupFiles = append(playgroundStartupFiles, unit)
	}

	return api.PlaygroundMachine{
		Name:         m.Name,
		Users:        m.Users,
		Resources:    m.Resources,
		StartupFiles: append(playgroundStartupFiles, m.StartupFiles...),
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

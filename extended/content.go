package extended

import (
	"slices"

	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/samber/lo"

	"github.com/sagikazarmark/labx/core"
)

type ContentManifest struct {
	Kind        content.ContentKind   `yaml:"kind"        json:"kind"`
	Title       string                `yaml:"title"       json:"title"`
	Description string                `yaml:"description" json:"description"`
	Channels    map[string]Channel    `yaml:"channels"    json:"channels"`
	Categories  []string              `yaml:"categories"  json:"categories"`
	Tags        []string              `yaml:"tagz"        json:"tagz"`
	CreatedAt   string                `yaml:"createdAt"   json:"createdAt"`
	UpdatedAt   string                `yaml:"updatedAt"   json:"updatedAt"`
	Cover       string                `yaml:"cover"       json:"cover"`
	Playground  ContentPlaygroundSpec `yaml:"playground"  json:"playground"`
	Tasks       map[string]Task       `yaml:"tasks"       json:"tasks"`

	// Challenge specific fields
	Difficulty string `yaml:"difficulty,omitempty" json:"difficulty,omitempty"`

	// Course specific fields
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	Slug string `yaml:"slug,omitempty" json:"slug,omitempty"`

	// Content embedding
	Challenges map[string]struct{}    `yaml:"challenges,omitempty" json:"challenges,omitempty"`
	Tutorials  map[string]struct{}    `yaml:"tutorials,omitempty"  json:"tutorials,omitempty"`
	Courses    map[string]CourseEmbed `yaml:"courses,omitempty"    json:"courses,omitempty"`

	// Training specific fields
	WorkingTitle string `yaml:"workingTitle,omitempty" json:"workingTitle,omitempty"`
}

func (m ContentManifest) Convert() core.ContentManifest {
	v := core.ContentManifest{
		Kind:        m.Kind,
		Title:       m.Title,
		Description: m.Description,
		Categories:  m.Categories,
		Tags:        m.Tags,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		Cover:       m.Cover,
		// Playground:  m.Playground.Convert(),
		Tasks: m.convertTasks(),

		Difficulty: m.Difficulty,

		Name: m.Name,
		Slug: m.Slug,

		Challenges: m.Challenges,
		Tutorials:  m.Tutorials,

		WorkingTitle: m.WorkingTitle,
	}

	if m.Kind != content.KindTraining && m.Kind != content.KindCourse {
		v.Playground = m.Playground.Convert()
	}

	return v
}

func (m ContentManifest) convertTasks() map[string]core.Task {
	tasks := map[string]core.Task{}

	for name, task := range m.Tasks {
		for _, machine := range task.Machine {
			for _, user := range task.User {
				newTask := task.ConvertCurrent(machine, user)

				// Dependency check and resolution
				for i, need := range newTask.Needs {
					// Dependency found with this name; need to check dependency resolution rules
					if dep, ok := m.Tasks[need]; ok {
						// Dependency must always run on the same machine
						if !slices.Contains(dep.Machine, machine) {
							panic("invalid dependency: machine")
						}

						// Dependency must have the same user in the list when running as multiple users
						if len(dep.User) > 1 && !slices.Contains(dep.User, user) {
							panic("invalid dependency: user")
						}

						newTask.Needs[i] = dep.currentName(need, machine, user)

						continue
					}

					// Dependency not found with this name so try a few other options
					// TODO: is this necessary?

					// Machine name AND user manually added
					if _, ok := m.Tasks[taskName(need, machine, user)]; ok {
						newTask.Needs[i] = taskName(need, machine, user)

						continue
					}

					// Machine name manually added
					if _, ok := m.Tasks[taskName(need, machine)]; ok {
						newTask.Needs[i] = taskName(need, machine)

						continue
					}

					// Task not found in content tasks; let's check the playground
					//
					// Machine name AND user
					if _, ok := m.Playground.Base.InitTasks[taskName(need, machine, user)]; ok {
						newTask.Needs[i] = taskName(need, machine, user)

						continue
					}

					// Machine name
					if _, ok := m.Playground.Base.InitTasks[taskName(need, machine)]; ok {
						newTask.Needs[i] = taskName(need, machine)

						continue
					}

					// Machine name
					if _, ok := m.Playground.Base.InitTasks[need]; ok {
						continue
					}

					// dependency not found anywhere
					panic("unknown dependency:" + need)
				}

				tasks[task.currentName(name, machine, user)] = newTask
			}
		}
	}

	return tasks
}

type ContentPlaygroundSpec struct {
	Name     string                  `yaml:"name"     json:"name"`
	Welcome  string                  `yaml:"welcome"  json:"welcome"`
	Networks []api.PlaygroundNetwork `yaml:"networks" json:"networks"`
	Machines PlaygroundMachines      `yaml:"machines" json:"machines"`
	Tabs     []api.PlaygroundTab     `yaml:"tabs"     json:"tabs"`

	BaseName string             `yaml:"-" json:"-"`
	Base     api.PlaygroundSpec `yaml:"-" json:"-"`
}

func (s ContentPlaygroundSpec) Convert() core.ContentPlaygroundSpec {
	return core.ContentPlaygroundSpec{
		Name:     s.Name,
		Networks: s.Networks,
		Machines: s.convertMachines(),
		Tabs:     s.Tabs,
	}
}

func (s ContentPlaygroundSpec) convertMachines() []core.ContentPlaygroundMachine {
	if s.BaseName == "flexbox" {
		return lo.Map(
			s.Machines.Convert(),
			func(machine api.PlaygroundMachine, _ int) core.ContentPlaygroundMachine {
				return core.ContentPlaygroundMachine{
					Name:         machine.Name,
					Users:        machine.Users,
					Kernel:       machine.Kernel,
					Drives:       machine.Drives,
					Network:      machine.Network,
					Resources:    machine.Resources,
					StartupFiles: machine.StartupFiles,
					NoSSH:        machine.NoSSH,
				}
			},
		)
	}

	parentMachines := lo.SliceToMap(
		s.Base.Machines,
		func(machine api.PlaygroundMachine) (string, api.PlaygroundMachine) {
			return machine.Name, machine
		},
	)

	// Make sure to include startup files from parent playground and apply welcome message
	machines := s.Machines.Convert()
	for i, machine := range machines {
		parentMachine := parentMachines[machine.Name]

		machines[i].StartupFiles = append(
			slices.Clone(parentMachine.StartupFiles),
			machine.StartupFiles...)

		// Apply welcome message to default users if specified
		if s.Welcome != "" {
			for j, user := range machine.Users {
				if user.Default && (user.Welcome == "" || user.Welcome == "-") {
					machines[i].Users[j].Welcome = s.Welcome
				}
			}
		}
	}

	return lo.Map(
		machines,
		func(machine api.PlaygroundMachine, _ int) core.ContentPlaygroundMachine {
			return core.ContentPlaygroundMachine{
				Name:         machine.Name,
				Users:        machine.Users,
				Kernel:       machine.Kernel,
				Drives:       machine.Drives,
				Network:      machine.Network,
				Resources:    machine.Resources,
				StartupFiles: machine.StartupFiles,
				NoSSH:        machine.NoSSH,
			}
		},
	)
}

type Task struct {
	Machine        StringList `yaml:"machine,omitempty" json:"machine,omitempty"`
	Init           bool       `yaml:"init"              json:"init"`
	User           StringList `yaml:"user"              json:"user"`
	TimeoutSeconds int        `yaml:"timeout_seconds"   json:"timeout_seconds"`
	Needs          []string   `yaml:"needs,omitempty"   json:"needs,omitempty"`
	Env            []string   `yaml:"env,omitempty"     json:"env,omitempty"`
	Run            string     `yaml:"run"               json:"run"`
}

func (t Task) Convert() core.Task {
	return core.Task{
		Init:           t.Init,
		TimeoutSeconds: t.TimeoutSeconds,
		Needs:          slices.Clone(t.Needs),
		Env:            slices.Clone(t.Env),
		Run:            t.Run,
	}
}

func (t Task) ConvertCurrent(machine string, user string) core.Task {
	task := t.Convert()
	task.Machine = machine
	task.User = user

	return task
}

func (t Task) currentName(name string, machine string, user string) string {
	var taskNameSegments []string

	if len(t.Machine) > 1 {
		taskNameSegments = append(taskNameSegments, machine)
	}

	if len(t.User) > 1 {
		taskNameSegments = append(taskNameSegments, user)
	}

	return taskName(name, taskNameSegments...)
}

type CourseEmbed struct {
	Lessons []string `yaml:"lessons,omitempty" json:"lessons,omitempty"`
}

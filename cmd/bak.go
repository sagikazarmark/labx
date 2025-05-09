package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
)

func fileInitTasks(fsys fs.FS, kind content.ContentKind, playgroundSpec api.PlaygroundSpec) ([]Task, error) {
	_, err := fs.Stat(fsys, fmt.Sprintf("dist/__static__/%s.tar.gz", kind.String()))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(playgroundSpec.Machines))

	targetDir := fmt.Sprintf("/opt/%s", kind)
	url := fmt.Sprintf("https://labs.iximiuz.com/__static__/%s.tar.gz?t=$(date +%%s)", kind)

	for _, machine := range playgroundSpec.Machines {
		name := "init_content_files_"

		if kind == content.KindPlayground {
			name = "init_files_"
		}

		name += toTaskName(machine.Name)

		task := Task{
			Name:    name,
			Machine: machine.Name,
			Init:    true,
			User:    "root",
			Run:     fmt.Sprintf("mkdir -p %s\nwget --no-cache -O - \"%s\" | tar -xz -C %s", targetDir, url, targetDir),
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

type Task struct {
	Name           string   `yaml:"name" json:"name"`
	Machine        string   `yaml:"machine,omitempty" json:"machine,omitempty"`
	Init           bool     `yaml:"init" json:"init"`
	User           string   `yaml:"user" json:"user"`
	TimeoutSeconds int      `yaml:"timeout_seconds" json:"timeout_seconds"`
	Needs          []string `yaml:"needs,omitempty" json:"needs,omitempty"`
	Run            string   `yaml:"run" json:"run"`
}

func toTaskName(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}

func taskKey(t Task) string { return t.Name }

func taskToInitTask(t Task) api.InitTask {
	return api.InitTask{
		Name:           t.Name,
		Machine:        t.Machine,
		Init:           t.Init,
		User:           t.User,
		TimeoutSeconds: t.TimeoutSeconds,
		Needs:          slices.Clone(t.Needs),
		Run:            t.Run,
	}
}

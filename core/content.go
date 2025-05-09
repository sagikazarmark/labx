package core

import (
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
)

type ContentManifest struct {
	Kind        content.ContentKind   `yaml:"kind" json:"kind"`
	Title       string                `yaml:"title" json:"title"`
	Description string                `yaml:"description" json:"description"`
	Categories  []string              `yaml:"categories" json:"categories"`
	Tags        []string              `yaml:"tagz" json:"tagz"`
	CreatedAt   string                `yaml:"createdAt" json:"createdAt"`
	UpdatedAt   string                `yaml:"updatedAt" json:"updatedAt"`
	Cover       string                `yaml:"cover" json:"cover"`
	Playground  ContentPlaygroundSpec `yaml:"playground" json:"playground"`
	Tasks       map[string]Task       `yaml:"tasks" json:"tasks"`
}

type ContentPlaygroundSpec struct {
	Name     string                  `yaml:"name,omitempty" json:"name,omitempty"`
	Machines []api.PlaygroundMachine `yaml:"machines,omitempty" json:"machines,omitempty"`
	Tabs     []api.PlaygroundTab     `yaml:"tabs,omitempty" json:"tabs,omitempty"`
}

type Task struct {
	Machine        string   `yaml:"machine,omitempty" json:"machine,omitempty"`
	Init           bool     `yaml:"init" json:"init"`
	User           string   `yaml:"user" json:"user"`
	TimeoutSeconds int      `yaml:"timeout_seconds" json:"timeout_seconds"`
	Needs          []string `yaml:"needs,omitempty" json:"needs,omitempty"`
	Run            string   `yaml:"run" json:"run"`
}

package core

import (
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
)

type ContentManifest struct {
	Kind        content.ContentKind   `yaml:"kind"                 json:"kind"`
	Title       string                `yaml:"title"                json:"title"`
	Description string                `yaml:"description"          json:"description"`
	Categories  []string              `yaml:"categories,omitempty" json:"categories,omitempty"`
	Tags        []string              `yaml:"tagz,omitempty"       json:"tagz,omitempty"`
	CreatedAt   string                `yaml:"createdAt"            json:"createdAt"`
	UpdatedAt   string                `yaml:"updatedAt,omitempty"  json:"updatedAt,omitempty"`
	Cover       string                `yaml:"cover,omitempty"      json:"cover,omitempty"`
	Playground  ContentPlaygroundSpec `yaml:"playground,omitempty" json:"playground,omitzero"`
	Tasks       map[string]Task       `yaml:"tasks,omitempty"      json:"tasks,omitzero"`

	// Challenge specific fields
	Difficulty string `yaml:"difficulty,omitempty" json:"difficulty,omitempty"`

	// Course specific fields
	Slug string `yaml:"slug,omitempty" json:"slug,omitempty"`
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
}

type ContentPlaygroundSpec struct {
	Name     string                  `yaml:"name,omitempty"     json:"name,omitempty"`
	Networks []api.PlaygroundNetwork `yaml:"networks"           json:"networks"`
	Machines []api.PlaygroundMachine `yaml:"machines,omitempty" json:"machines,omitempty"`
	Tabs     []api.PlaygroundTab     `yaml:"tabs,omitempty"     json:"tabs,omitempty"`
}

type Task struct {
	Machine        string   `yaml:"machine,omitempty" json:"machine,omitempty"`
	Init           bool     `yaml:"init"              json:"init"`
	User           string   `yaml:"user"              json:"user"`
	TimeoutSeconds int      `yaml:"timeout_seconds"   json:"timeout_seconds"`
	Needs          []string `yaml:"needs,omitempty"   json:"needs,omitempty"`
	Env            []string `yaml:"env,omitempty"     json:"env,omitempty"`
	Run            string   `yaml:"run"               json:"run"`
}

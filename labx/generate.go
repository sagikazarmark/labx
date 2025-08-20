package labx

import (
	"fmt"
	"io/fs"
	"os"
	"text/template"

	"github.com/goccy/go-yaml"
)

// manifestKind represents a minimal manifest structure to determine routing
type manifestKind struct {
	Kind string `yaml:"kind" json:"kind"`
}

// GenerateOpts contains options for the Generate function
type GenerateOpts struct {
	Root         *os.Root
	Output       *os.Root
	Channel      string
	TemplateDirs []fs.FS
	DataDirs     []fs.FS
}

// GenerateContext contains the parsed state for content generation
type GenerateContext struct {
	Root         *os.Root
	Output       *os.Root
	Channel      string
	BaseTemplate *template.Template
	ExtraData    map[string]any
}

// Generate processes content based on the manifest kind, routing to appropriate handlers
func Generate(opts GenerateOpts) error {
	// Read and parse just the kind field from manifest.yaml
	manifestFile, err := opts.Root.FS().Open("manifest.yaml")
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	decoder := yaml.NewDecoder(manifestFile)

	var kind manifestKind
	err = decoder.Decode(&kind)
	if err != nil {
		return err
	}

	// Parse global templates
	baseTemplate, err := createBaseTemplate(opts.Root.FS(), opts.TemplateDirs)
	if err != nil {
		return fmt.Errorf("create global templates: %w", err)
	}

	// Load extra template data once
	extraData, err := loadAllExtraData(opts.Root.FS(), opts.DataDirs)
	if err != nil {
		return fmt.Errorf("load extra template data: %w", err)
	}

	// Create the context with shared state
	ctx := GenerateContext{
		Root:         opts.Root,
		Output:       opts.Output,
		Channel:      opts.Channel,
		BaseTemplate: baseTemplate,
		ExtraData:    extraData,
	}

	// Route based on kind
	if kind.Kind == "playground" {
		return Playground(ctx)
	}

	// Everything else goes through content processing
	return Content(ctx)
}

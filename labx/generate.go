package labx

import (
	"fmt"
	"io/fs"
	"os"
	"text/template"
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
	kind, ctx, err := prepareGenerateContext(opts)
	if err != nil {
		return err
	}

	// Route based on kind
	if kind.Kind == "playground" {
		return Playground(ctx)
	}

	// Everything else goes through content processing
	return Content(ctx)
}

func prepareGenerateContext(opts GenerateOpts) (manifestKind, GenerateContext, error) {
	kind, err := loadYAMLFile[manifestKind](opts.Root.FS(), "manifest.yaml")
	if err != nil {
		return manifestKind{}, GenerateContext{}, err
	}

	baseTemplate, err := createBaseTemplate(opts.Root.FS(), opts.TemplateDirs)
	if err != nil {
		return manifestKind{}, GenerateContext{}, fmt.Errorf("create global templates: %w", err)
	}

	extraData, err := loadAllExtraData(opts.Root.FS(), opts.DataDirs)
	if err != nil {
		return manifestKind{}, GenerateContext{}, fmt.Errorf("load extra template data: %w", err)
	}

	ctx := GenerateContext{
		Root:         opts.Root,
		Output:       opts.Output,
		Channel:      opts.Channel,
		BaseTemplate: baseTemplate,
		ExtraData:    extraData,
	}

	return kind, ctx, nil
}

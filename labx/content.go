package labx

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/samber/lo"

	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/extended"
)

func Content(ctx GenerateContext) error {
	extendedManifest, err := loadContentManifest(ctx.Root.FS(), ctx.Channel)
	if err != nil {
		return err
	}

	manifest := extendedManifest.Convert()

	indexFile, err := ctx.Output.Create("index.md")
	if err != nil {
		return err
	}
	defer indexFile.Close()

	err = writeManifest(indexFile, manifest)
	if err != nil {
		return err
	}

	if strings.ToLower(ctx.Channel) == "beta" {
		_, err = io.WriteString(indexFile, betaNotice)
		if err != nil {
			return err
		}
	}

	// Copy global templates and add local content templates
	tpl, err := createContentTemplateFromGlobal(ctx.BaseTemplate, ctx.Root.FS())
	if err != nil {
		return err
	}

	renderCtx := renderContext{
		Root:         ctx.Root,
		Output:       ctx.Output,
		Channel:      ctx.Channel,
		Name:         extendedManifest.Channels[ctx.Channel].Name,
		Manifest:     manifest,
		Extra:        ctx.ExtraData,
		BaseTemplate: ctx.BaseTemplate,
	}

	data := templateData{
		Channel:  ctx.Channel,
		Name:     renderCtx.Name,
		Manifest: manifest,
		Extra:    ctx.ExtraData,
	}

	err = tpl.ExecuteTemplate(indexFile, "index.md", data)
	if err != nil {
		return err
	}

	// Copy static files if they exist at the root level
	hasStatic, err := dirExists(ctx.Root.FS(), "static")
	if err != nil {
		return err
	}

	if hasStatic {
		err = copyStaticFiles(ctx.Root, ctx.Output, "static", "__static__")
		if err != nil {
			return fmt.Errorf("copy static files: %w", err)
		}
	}

	// Handle content-specific rendering
	switch manifest.Kind {
	case content.KindChallenge:
		err := renderChallenge(renderCtx, tpl)
		if err != nil {
			return err
		}
	case content.KindCourse:
		err := renderCourse(renderCtx)
		if err != nil {
			return err
		}
	case content.KindTraining:
		err := renderTraining(renderCtx, tpl)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadContentManifest(fsys fs.FS, channel string) (extended.ContentManifest, error) {
	manifestFile, err := fsys.Open("manifest.yaml")
	if err != nil {
		return extended.ContentManifest{}, err
	}
	defer manifestFile.Close()

	decoder := yaml.NewDecoder(manifestFile)

	var extendedManifest extended.ContentManifest

	err = decoder.Decode(&extendedManifest)
	if err != nil {
		return extended.ContentManifest{}, err
	}

	if extendedManifest.Playground.Name != "" {
		hf, err := hasFiles(fsys, extendedManifest.Kind)
		if err != nil {
			return extended.ContentManifest{}, err
		}

		basePlayground, err := getPlaygroundManifest(extendedManifest.Playground.Name)
		if err != nil {
			return extended.ContentManifest{}, err
		}

		if hf {
			machines := lo.Map(
				extendedManifest.Playground.Machines,
				func(machine extended.PlaygroundMachine, _ int) string {
					return machine.Name
				},
			)

			if len(machines) == 0 {
				machines = lo.Map(
					basePlayground.Playground.Machines,
					func(machine api.PlaygroundMachine, _ int) string {
						return machine.Name
					},
				)
			}

			const name = "init_content_files"

			extendedManifest.Tasks[name] = extended.Task{
				Machine: machines,
				Init:    true,
				User:    extended.StringList{"root"},
				Run:     createDownloadScript(extendedManifest.Kind),
			}
		}

		extendedManifest.Playground.BaseName = basePlayground.Name
		extendedManifest.Playground.Base = basePlayground.Playground

		machinesProcessor := MachinesProcessor{
			MachineProcessor: MachineProcessor{
				UserProcessor: MachineUserProcessor{
					Fsys: fsys,
				},
				DriveProcessor: MachineDriveProcessor{
					ContentKind:      extendedManifest.Kind,
					ContentName:      "",
					Channel:          channel,
					DefaultImageRepo: defaultImageRepo,
				},
				StartupFileProcessor: MachineStartupFileProcessor{
					Fsys: fsys,
				},
			},
		}

		machines, err := machinesProcessor.Process(extendedManifest.Playground.Machines)
		if err != nil {
			return extended.ContentManifest{}, err
		}

		extendedManifest.Playground.Machines = machines
	}

	// Apply channel-specific title processing only for real content kinds (not lessons)
	if channel != "live" && string(extendedManifest.Kind) != "lesson" {
		extendedManifest.Title = fmt.Sprintf(
			"%s: %s",
			strings.ToUpper(channel),
			extendedManifest.Title,
		)
	}

	return extendedManifest, err
}

func convertContentManifest(fsys fs.FS, channel string) (core.ContentManifest, error) {
	extendedManifest, err := loadContentManifest(fsys, channel)

	manifest := extendedManifest.Convert()

	return manifest, err
}

// renderContext holds all the data needed for rendering templates
type renderContext struct {
	Root         *os.Root
	Output       *os.Root
	Channel      string
	Name         string
	Manifest     core.ContentManifest
	Extra        map[string]any
	BaseTemplate *template.Template
}

// templateData holds the data passed to template executions
type templateData struct {
	Channel  string
	Name     string
	Manifest core.ContentManifest
	Extra    map[string]any
}

// frontMatterWriter automatically adds front matter delimiters on first write
type frontMatterWriter struct {
	writer     io.Writer
	firstWrite bool
}

// newFrontMatterWriter creates a new frontMatterWriter
func newFrontMatterWriter(writer io.Writer) *frontMatterWriter {
	return &frontMatterWriter{writer: writer}
}

func (w *frontMatterWriter) Write(p []byte) (n int, err error) {
	if !w.firstWrite {
		w.firstWrite = true

		// Write opening delimiter
		_, err = io.WriteString(w.writer, "---\n")
		if err != nil {
			return 0, err
		}

		// Write the content
		n, err = w.writer.Write(p)
		if err != nil {
			return n, err
		}

		// Write closing delimiter
		_, err = io.WriteString(w.writer, "---\n")
		if err != nil {
			return n, err
		}

		return n, nil
	}

	// Subsequent writes go directly to the underlying writer
	return w.writer.Write(p)
}

// createContentTemplateFromGlobal creates a content template by copying global templates and adding local ones
func createContentTemplateFromGlobal(
	globalTpl *template.Template,
	fsys fs.FS,
) (*template.Template, error) {
	// Clone the global template to avoid conflicts
	tpl, err := globalTpl.Clone()
	if err != nil {
		return nil, fmt.Errorf("clone global template: %w", err)
	}

	// Parse local content-level patterns (these can override global templates)
	patterns := []string{
		"*.md",
		"templates/*.md",
	}

	return parseTemplatePatterns(tpl, fsys, patterns)
}

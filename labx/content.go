package labx

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/iximiuz/labctl/content"

	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/extended"
)

func Content(ctx GenerateContext) error {
	extendedManifest, err := loadContentManifest(ctx.Root.FS(), ctx.Channel)
	if err != nil {
		return err
	}

	manifest := extendedManifest.Convert()

	// Copy global templates and add local content templates
	tpl, err := createContentTemplateFromGlobal(ctx.BaseTemplate, ctx.Root.FS())
	if err != nil {
		return err
	}

	renderCtx := newRenderContext(ctx, manifest, extendedManifest.Channels[ctx.Channel].Name)

	err = renderContentIndex(renderCtx, tpl, strings.ToLower(ctx.Channel) == "beta")
	if err != nil {
		return err
	}

	err = copyStaticFilesIfExists(ctx.Root, ctx.Output, "static", "__static__")
	if err != nil {
		return fmt.Errorf("copy static files: %w", err)
	}

	return renderContentTemplates(renderCtx, tpl)
}

func loadContentManifest(fsys fs.FS, channel string) (extended.ContentManifest, error) {
	extendedManifest, err := loadYAMLFile[extended.ContentManifest](fsys, "manifest.yaml")
	if err != nil {
		return extended.ContentManifest{}, err
	}

	if extendedManifest.Playground.Name != "" {
		basePlayground, err := getPlaygroundManifest(extendedManifest.Playground.Name)
		if err != nil {
			return extended.ContentManifest{}, err
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

func newRenderContext(
	ctx GenerateContext,
	manifest core.ContentManifest,
	name string,
) renderContext {
	return renderContext{
		Root:         ctx.Root,
		Output:       ctx.Output,
		Channel:      ctx.Channel,
		Name:         name,
		Manifest:     manifest,
		Extra:        ctx.ExtraData,
		BaseTemplate: ctx.BaseTemplate,
	}
}

func newTemplateData(ctx renderContext) templateData {
	return templateData{
		Channel:  ctx.Channel,
		Name:     ctx.Name,
		Manifest: ctx.Manifest,
		Extra:    ctx.Extra,
	}
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

func renderContentIndex(
	ctx renderContext,
	tpl *template.Template,
	includeBetaNotice bool,
) error {
	indexFile, err := createOutputFile(ctx.Output, "index.md")
	if err != nil {
		return err
	}
	defer indexFile.Close()

	err = writeManifest(indexFile, ctx.Manifest)
	if err != nil {
		return err
	}

	if includeBetaNotice {
		_, err = io.WriteString(indexFile, betaNotice)
		if err != nil {
			return err
		}
	}

	return tpl.ExecuteTemplate(indexFile, "index.md", newTemplateData(ctx))
}

func renderContentTemplates(ctx renderContext, tpl *template.Template) error {
	switch ctx.Manifest.Kind {
	case content.KindChallenge:
		return renderChallenge(ctx, tpl)
	case content.KindCourse:
		return renderCourse(ctx)
	case content.KindTraining:
		return renderTraining(ctx, tpl)
	}

	return nil
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

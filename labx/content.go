package labx

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/go-sprout/sprout"
	sproutstrings "github.com/go-sprout/sprout/registry/strings"
	sprouttime "github.com/go-sprout/sprout/registry/time"
	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/samber/lo"

	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/extended"
	"github.com/sagikazarmark/labx/pkg/sproutx"
)

func Content(root *os.Root, output *os.Root, channel string) error {
	manifest, err := convertContentManifest(root.FS(), channel)
	if err != nil {
		return err
	}

	indexFile, err := output.Create("index.md")
	if err != nil {
		return err
	}
	defer indexFile.Close()

	err = writeManifest(indexFile, manifest)
	if err != nil {
		return err
	}

	if strings.ToLower(channel) == "beta" {
		_, err = io.WriteString(indexFile, betaNotice)
		if err != nil {
			return err
		}
	}

	tpl, err := createContentTemplate(root.FS())
	if err != nil {
		return err
	}

	extraData, err := loadExtraTemplateData(root.FS())
	if err != nil {
		return fmt.Errorf("load extra template data: %w", err)
	}

	ctx := renderContext{
		Root:     root,
		Output:   output,
		Channel:  channel,
		Manifest: manifest,
		Extra:    extraData,
	}

	data := templateData{
		Channel:  ctx.Channel,
		Manifest: ctx.Manifest,
		Extra:    ctx.Extra,
	}

	err = tpl.ExecuteTemplate(indexFile, "index.md", data)
	if err != nil {
		return err
	}

	// Copy static files if they exist at the root level
	hasStatic, err := dirExists(root.FS(), "static")
	if err != nil {
		return err
	}

	if hasStatic {
		err = copyStaticFiles(root, output, "static", "__static__")
		if err != nil {
			return fmt.Errorf("copy static files: %w", err)
		}
	}

	// Handle content-specific rendering
	switch manifest.Kind {
	case content.KindChallenge:
		err := renderChallenge(ctx, tpl)
		if err != nil {
			return err
		}
	case content.KindCourse:
		err := renderCourse(ctx)
		if err != nil {
			return err
		}
	case content.KindTraining:
		err := renderTraining(ctx, tpl)
		if err != nil {
			return err
		}
	}

	return nil
}

func convertContentManifest(fsys fs.FS, channel string) (core.ContentManifest, error) {
	manifestFile, err := fsys.Open("manifest.yaml")
	if err != nil {
		return core.ContentManifest{}, err
	}
	defer manifestFile.Close()

	decoder := yaml.NewDecoder(manifestFile)

	var extendedManifest extended.ContentManifest

	err = decoder.Decode(&extendedManifest)
	if err != nil {
		return core.ContentManifest{}, err
	}

	if extendedManifest.Playground.Name != "" {
		hf, err := hasFiles(fsys, extendedManifest.Kind)
		if err != nil {
			return core.ContentManifest{}, err
		}

		basePlayground, err := getPlaygroundManifest(extendedManifest.Playground.Name)
		if err != nil {
			return core.ContentManifest{}, err
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
			return core.ContentManifest{}, err
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

	manifest := extendedManifest.Convert()

	return manifest, err
}

// renderContext holds all the data needed for rendering templates
type renderContext struct {
	Root     *os.Root
	Output   *os.Root
	Channel  string
	Manifest core.ContentManifest
	Extra    map[string]any
}

// templateData holds the data passed to template executions
type templateData struct {
	Channel  string
	Manifest core.ContentManifest
	Extra    map[string]any
}

// copyStaticFiles copies static files from source to destination
func copyStaticFiles(root *os.Root, output *os.Root, sourcePath, destPath string) error {
	fsys := root.FS()

	// Create the parent static directory first
	err := output.Mkdir(destPath, 0o755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	return fs.WalkDir(fsys, sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourcePath {
			return nil
		}

		// Calculate relative path from source
		relPath := strings.TrimPrefix(path, sourcePath+"/")
		outputPath := destPath + "/" + relPath

		if d.IsDir() {
			// Create directory in destination
			err = output.Mkdir(outputPath, 0o755)
			if err != nil && !os.IsExist(err) {
				return err
			}
			return nil
		}

		// Copy file
		sourceFile, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		destFile, err := output.Create(outputPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, sourceFile)
		return err
	})
}

// dirExists checks if a directory exists
func dirExists(fsys fs.FS, path string) (bool, error) {
	stat, err := fs.Stat(fsys, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return stat.IsDir(), nil
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

// createTemplateFuncs creates template functions for the given filesystem
func createTemplateFuncs(fsys fs.FS) template.FuncMap {
	return sprout.New(
		sprout.WithRegistries(
			sproutstrings.NewRegistry(),
			sproutx.NewFSRegistry(fsys),
			sproutx.NewStringsRegistry(),
			sprouttime.NewRegistry(),
		),
	).Build()
}

// parseTemplatePatterns parses template patterns from a filesystem into a template
func parseTemplatePatterns(
	tpl *template.Template,
	fsys fs.FS,
	patterns []string,
) (*template.Template, error) {
	for _, pattern := range patterns {
		matches, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, err
		}

		if len(matches) == 0 {
			continue
		}

		tpl, err = tpl.ParseFS(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("parse templates with pattern %s: %w", pattern, err)
		}
	}
	return tpl, nil
}

// createContentTemplate creates a template instance for content-level rendering
func createContentTemplate(fsys fs.FS) (*template.Template, error) {
	tplFuncs := createTemplateFuncs(fsys)
	tpl := template.New("").Funcs(tplFuncs)

	// Parse content-level patterns
	patterns := []string{
		"*.md",
		"templates/*.md",
	}

	return parseTemplatePatterns(tpl, fsys, patterns)
}

package labx

import (
	"bytes"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/sagikazarmark/go-finder"

	"github.com/sagikazarmark/labx/extended"
)

func Playground(ctx GenerateContext) error {
	manifest, err := convertPlaygroundManifest(
		ctx.Root.FS(),
		ctx.Channel,
		ctx.BaseTemplate,
		ctx.ExtraData,
	)
	if err != nil {
		return err
	}

	if strings.ToLower(ctx.Channel) == "beta" {
		manifest.Markdown = betaNotice + manifest.Markdown
	}

	// Create the manifest.yaml file
	err = renderManifest(ctx.Output, "manifest.yaml", manifest)
	if err != nil {
		return err
	}

	// Copy static files if they exist
	hasStatic, err := dirExists(ctx.Root.FS(), "static")
	if err != nil {
		return err
	}

	if hasStatic {
		err = copyStaticFiles(ctx.Root, ctx.Output, "static", "__static__")
		if err != nil {
			return err
		}
	}

	return nil
}

func convertPlaygroundManifest(
	fsys fs.FS,
	channel string,
	baseTemplate *template.Template,
	extraData map[string]any,
) (api.PlaygroundManifest, error) {
	manifestFile, err := fsys.Open("manifest.yaml")
	if err != nil {
		return api.PlaygroundManifest{}, err
	}
	defer manifestFile.Close()

	decoder := yaml.NewDecoder(manifestFile)

	var extendedManifest extended.PlaygroundManifest

	err = decoder.Decode(&extendedManifest)
	if err != nil {
		return api.PlaygroundManifest{}, err
	}

	// basePlayground, err := getPlaygroundManifest(extendedManifest.Base)
	// if err != nil {
	// 	return api.PlaygroundManifest{}, err
	// }

	extendedManifest.Playground.BaseName = extendedManifest.Base

	playgroundProcessor := PlaygroundProcessor{
		Channel: channel,
		Fsys:    fsys,
		MachinesProcessor: MachinesProcessor{
			MachineProcessor: MachineProcessor{
				UserProcessor: MachineUserProcessor{
					Fsys: fsys,
				},
				DriveProcessor: MachineDriveProcessor{
					ContentKind:      content.KindPlayground,
					ContentName:      extendedManifest.Name,
					Channel:          channel,
					DefaultImageRepo: defaultImageRepo,
				},
				StartupFileProcessor: MachineStartupFileProcessor{
					Fsys: fsys,
				},
			},
		},
	}

	extendedManifest, err = playgroundProcessor.Process(extendedManifest)
	if err != nil {
		return api.PlaygroundManifest{}, err
	}

	manifest := extendedManifest.Convert()

	if manifest.Markdown == "" {
		markdown, err := readAndRenderMarkdown(fsys, channel, manifest, baseTemplate, extraData)
		if err != nil {
			return manifest, err
		}

		manifest.Markdown = markdown
	}

	return manifest, err
}

func readAndRenderMarkdown(
	fsys fs.FS,
	channel string,
	manifest api.PlaygroundManifest,
	baseTemplate *template.Template,
	extraData map[string]any,
) (string, error) {
	finder := finder.Finder{
		Paths: []string{""},
		Names: []string{"README.md", "manifest.md"},
		Type:  finder.FileTypeFile,
	}

	markdownFile, err := finder.Find(fsys)
	if err != nil {
		return "", err
	}

	if len(markdownFile) == 0 {
		return "", nil
	}

	templateName := markdownFile[0]

	// Create template by copying global template and adding playground-specific ones
	tpl, err := createPlaygroundTemplate(fsys, baseTemplate)
	if err != nil {
		return "", fmt.Errorf("create playground template: %w", err)
	}

	// Create template data for playground
	data := playgroundTemplateData{
		Channel:  channel,
		Manifest: manifest,
		Extra:    extraData,
	}

	// Execute the template
	var buf bytes.Buffer
	err = tpl.ExecuteTemplate(&buf, templateName, data)
	if err != nil {
		return "", fmt.Errorf("execute markdown template %s: %w", templateName, err)
	}

	return buf.String(), nil
}

// playgroundTemplateData holds the data passed to playground template executions
type playgroundTemplateData struct {
	Channel  string
	Manifest api.PlaygroundManifest
	Extra    map[string]any
}

// createPlaygroundTemplate creates a playground template by copying global templates and adding local ones
func createPlaygroundTemplate(
	fsys fs.FS,
	baseTemplate *template.Template,
) (*template.Template, error) {
	// Clone the global template to avoid conflicts
	tpl, err := baseTemplate.Clone()
	if err != nil {
		return nil, fmt.Errorf("clone global template: %w", err)
	}

	// Parse local playground-level patterns (these can override global templates)
	patterns := []string{
		"manifest.md",
		"README.md",
		"templates/*.md",
		"templates/**/*.md",
	}

	return parseTemplatePatterns(tpl, fsys, patterns)
}

func getPlaygroundManifest(name string) (api.PlaygroundManifest, error) {
	var b bytes.Buffer

	cmd := exec.Command("labctl", "playground", "manifest", name)
	cmd.Stdout = &b

	if err := cmd.Run(); err != nil {
		return api.PlaygroundManifest{}, err
	}

	decoder := yaml.NewDecoder(&b)

	var manifest api.PlaygroundManifest

	err := decoder.Decode(&manifest)
	if err != nil {
		return api.PlaygroundManifest{}, err
	}

	return manifest, nil
}

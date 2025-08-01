package labx

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/sagikazarmark/go-finder"

	"github.com/sagikazarmark/labx/extended"
)

func Playground(root *os.Root, output *os.Root, channel string) error {
	manifest, err := convertPlaygroundManifest(root.FS(), channel)
	if err != nil {
		return err
	}

	if strings.ToLower(channel) == "beta" {
		manifest.Markdown = betaNotice + manifest.Markdown
	}

	// Create the manifest.yaml file
	err = renderManifest(output, "manifest.yaml", manifest)
	if err != nil {
		return err
	}

	// Copy static files if they exist
	hasStatic, err := dirExists(root.FS(), "static")
	if err != nil {
		return err
	}

	if hasStatic {
		err = copyStaticFiles(root, output, "static", "__static__")
		if err != nil {
			return err
		}
	}

	return nil
}

func convertPlaygroundManifest(fsys fs.FS, channel string) (api.PlaygroundManifest, error) {
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

	basePlayground, err := getPlaygroundManifest(extendedManifest.Base)
	if err != nil {
		return api.PlaygroundManifest{}, err
	}

	extendedManifest.Playground.BaseName = basePlayground.Name
	extendedManifest.Playground.Base = basePlayground.Playground

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
		markdown, err := readAndRenderMarkdown(fsys, channel, manifest)
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
) (string, error) {
	var templateName string

	finder := finder.Finder{
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

	// Create template and render
	tpl, err := createPlaygroundTemplate(fsys)
	if err != nil {
		return "", fmt.Errorf("create playground template: %w", err)
	}

	// Load extra template data
	extraData, err := loadExtraTemplateData(fsys)
	if err != nil {
		return "", fmt.Errorf("load extra template data: %w", err)
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

// createPlaygroundTemplate creates a template instance for playground-level rendering
func createPlaygroundTemplate(fsys fs.FS) (*template.Template, error) {
	tplFuncs := createTemplateFuncs(fsys)
	tpl := template.New("").Funcs(tplFuncs)

	// Parse playground-level patterns
	patterns := []string{
		"manifest.md",
		"README.md",
		"templates/*.md",
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

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
	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/extended"
	"github.com/samber/lo"
)

func Content(fsys fs.FS, channel string) error {
	manifest, err := convertContentManifest(fsys, channel)
	if err != nil {
		return err
	}

	indexFile, err := os.Create("dist/index.md")
	if err != nil {
		return err
	}
	defer indexFile.Close()

	encoder := yaml.NewEncoder(
		indexFile,
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.IndentSequence(true),
	)

	_, err = io.WriteString(indexFile, "---\n")
	if err != nil {
		return err
	}

	err = encoder.Encode(manifest)
	if err != nil {
		return err
	}

	_, err = io.WriteString(indexFile, "---\n")
	if err != nil {
		return err
	}

	if strings.ToLower(channel) == "beta" {
		_, err = io.WriteString(indexFile, betaNotice)
		if err != nil {
			return err
		}
	}

	tplData := templateData{fsys}

	tpl, err := template.New("index.md").Funcs(tplFuncs).ParseFS(fsys, "index.md")
	if err != nil {
		return err
	}

	err = tpl.Execute(indexFile, tplData)
	if err != nil {
		return err
	}

	if manifest.Kind == content.KindChallenge {
		hasSolution, err := fileExists(fsys, "solution.md")
		if err != nil {
			return err
		}

		if hasSolution {
			solutionFile, err := os.Create("dist/solution.md")
			if err != nil {
				return err
			}
			defer solutionFile.Close()

			tpl, err := template.New("solution.md").Funcs(tplFuncs).ParseFS(fsys, "solution.md")
			if err != nil {
				return err
			}

			err = tpl.Execute(solutionFile, tplData)
			if err != nil {
				return err
			}
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

	hf, err := hasFiles(fsys, extendedManifest.Kind)
	if err != nil {
		return core.ContentManifest{}, err
	}

	basePlayground, err := getPlaygroundManifest(extendedManifest.Playground.Name)
	if err != nil {
		return core.ContentManifest{}, err
	}

	if hf {
		machines := lo.Map(extendedManifest.Playground.Machines, func(machine extended.PlaygroundMachine, _ int) string {
			return machine.Name
		})

		if len(machines) == 0 {
			machines = lo.Map(basePlayground.Playground.Machines, func(machine api.PlaygroundMachine, _ int) string {
				return machine.Name
			})
		}

		const name = "init_content_files"

		extendedManifest.Tasks[name] = extended.Task{
			Machine: machines,
			Init:    true,
			User:    extended.StringList{"root"},
			Run:     createDownloadScript(extendedManifest.Kind),
		}
	}

	if channel != "live" {
		extendedManifest.Title = fmt.Sprintf("%s: %s", strings.ToUpper(channel), extendedManifest.Title)
	}

	extendedManifest.Playground.Base = basePlayground.Playground

	manifest := extendedManifest.Convert()

	return manifest, err
}

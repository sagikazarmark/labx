package labx

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/extended"
	"github.com/samber/lo"
)

func Content(fsys fs.FS, channel string) (core.ContentManifest, error) {
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

	if hf {
		machines := lo.Map(extendedManifest.Playground.Machines, func(machine extended.PlaygroundMachine, _ int) string {
			return machine.Name
		})

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

	basePlayground, err := getPlaygroundManifest(extendedManifest.Playground.Name)
	if err != nil {
		return core.ContentManifest{}, err
	}

	extendedManifest.Playground.Base = basePlayground.Playground

	manifest := extendedManifest.Convert()

	return manifest, err
}

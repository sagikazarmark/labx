package labx

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/sagikazarmark/labx/extended"
	"github.com/samber/lo"
)

func Playground(fsys fs.FS, channel string) (api.PlaygroundManifest, error) {
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

	hf, err := hasFiles(fsys, content.KindPlayground)
	if err != nil {
		return api.PlaygroundManifest{}, err
	}

	basePlayground, err := getPlaygroundManifest(extendedManifest.Base)
	if err != nil {
		return api.PlaygroundManifest{}, err
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

		const name = "init_files"

		extendedManifest.Playground.InitTasks[name] = extended.InitTask{
			Name:    name,
			Machine: machines,
			Init:    true,
			User:    extended.StringList{"root"},
			Run:     createDownloadScript(content.KindPlayground),
		}
	}

	if channel != "live" {
		extendedManifest.Title = fmt.Sprintf("%s: %s", strings.ToUpper(channel), extendedManifest.Title)
	}

	channelData, ok := extendedManifest.Channels[channel]
	if !ok {
		return api.PlaygroundManifest{}, errors.New("missing channel data: " + channel)
	}

	extendedManifest.Name = channelData.Name

	if channelData.Public {
		extendedManifest.Playground.AccessControl = api.PlaygroundAccessControl{
			CanList:  []string{"anyone"},
			CanRead:  []string{"anyone"},
			CanStart: []string{"anyone"},
		}
	}

	extendedManifest.Playground.BaseName = basePlayground.Name
	extendedManifest.Playground.Base = basePlayground.Playground

	manifest := extendedManifest.Convert()

	if manifest.Markdown == "" {
		markdown, err := readMarkdown(fsys)
		if err != nil {
			return manifest, err
		}

		manifest.Markdown = markdown
	}

	return manifest, err
}

func readMarkdown(fsys fs.FS) (string, error) {
	content, err := fs.ReadFile(fsys, "manifest.md")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	} else if err == nil {
		return string(content), nil
	}

	content, err = fs.ReadFile(fsys, "README.md")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	} else if err == nil {
		return string(content), nil
	}

	return "", nil
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

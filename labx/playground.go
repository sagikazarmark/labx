package labx

import (
	"bytes"
	"errors"
	"io/fs"
	"os/exec"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"

	"github.com/sagikazarmark/labx/extended"
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

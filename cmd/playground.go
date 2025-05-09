package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/sagikazarmark/labx/xapi"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

type playgroundOptions struct {
	path    string
	channel string
}

func NewPlaygroundCommand() *cobra.Command {
	var opts playgroundOptions

	cmd := &cobra.Command{
		Use:   "playground",
		Short: "Generate playground content",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlayground(&opts)
		},
	}

	flags := cmd.Flags()

	flags.StringVar(
		&opts.path,
		"path",
		".",
		`Path to load manifest from`,
	)

	flags.StringVar(
		&opts.channel,
		"channel",
		"dev",
		`Which channel to push the playground to`,
	)

	return cmd
}

func runPlayground(opts *playgroundOptions) error {
	fsys, err := os.OpenRoot(opts.path)
	if err != nil {
		return err
	}

	manifest, err := playground(fsys.FS(), opts.channel)
	if err != nil {
		panic(err)
	}

	bytes, err := yaml.MarshalWithOptions(
		manifest,
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.IndentSequence(true),
	)
	if err != nil {
		return err
	}

	fmt.Println(string(bytes))

	return nil
}

func playground(fsys fs.FS, channel string) (api.PlaygroundManifest, error) {
	manifestFile, err := fsys.Open("manifest.yaml")
	if err != nil {
		return api.PlaygroundManifest{}, err
	}

	decoder := yaml.NewDecoder(manifestFile)

	var sourceManifest xapi.PlaygroundManifest

	err = decoder.Decode(&sourceManifest)
	if err != nil {
		return api.PlaygroundManifest{}, err
	}

	hf, err := hasFiles(fsys, content.KindPlayground)
	if err != nil {
		return api.PlaygroundManifest{}, err
	}

	if hf {
		machines := lo.Map(sourceManifest.Playground.Machines, func(machine xapi.PlaygroundMachine, _ int) string {
			return machine.Name
		})

		const name = "init_files"

		sourceManifest.Playground.InitTasks[name] = xapi.InitTask{
			Name:    name,
			Machine: machines,
			Init:    true,
			User:    xapi.StringList{"root"},
			Run:     createDownloadScript(content.KindPlayground),
		}
	}

	if channel != "live" {
		sourceManifest.Title = fmt.Sprintf("%s: %s", strings.ToUpper(channel), sourceManifest.Title)
	}

	// TODO: channel access control

	basePlayground, err := getPlaygroundManifest(sourceManifest.Base)
	if err != nil {
		return api.PlaygroundManifest{}, err
	}

	sourceManifest.Playground.Base = basePlayground.Playground

	manifest := sourceManifest.Convert()

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

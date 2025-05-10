package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/extended"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

type contentOptions struct {
	path    string
	channel string
}

func NewContentCommand() *cobra.Command {
	var opts contentOptions

	cmd := &cobra.Command{
		Use:   "content",
		Short: "Generate content",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContent(&opts)
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

func runContent(opts *contentOptions) error {
	fsys, err := os.OpenRoot(opts.path)
	if err != nil {
		return err
	}

	manifest, err := _content(fsys.FS(), opts.channel)
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

func _content(fsys fs.FS, channel string) (core.ContentManifest, error) {
	manifestFile, err := fsys.Open("manifest.yaml")
	if err != nil {
		return core.ContentManifest{}, err
	}

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

	// TODO: channel access control

	basePlayground, err := getPlaygroundManifest(extendedManifest.Playground.Name)
	if err != nil {
		return core.ContentManifest{}, err
	}

	extendedManifest.Playground.Base = basePlayground.Playground

	manifest := extendedManifest.Convert()

	return manifest, err
}

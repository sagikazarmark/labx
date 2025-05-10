package cmd

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/sagikazarmark/labx/labx"
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

	manifest, err := labx.Playground(fsys.FS(), opts.channel)
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

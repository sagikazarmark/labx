package cmd

import (
	"io"
	"os"
	"strings"

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
			return runPlayground(&opts, cmd.OutOrStdout())
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

func runPlayground(opts *playgroundOptions, output io.Writer) error {
	fsys, err := os.OpenRoot(opts.path)
	if err != nil {
		return err
	}

	manifest, err := labx.Playground(fsys.FS(), opts.channel)
	if err != nil {
		return err
	}

	if strings.ToLower(opts.channel) == "beta" {
		manifest.Markdown = betaNotice + manifest.Markdown
	}

	encoder := yaml.NewEncoder(
		output,
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.IndentSequence(true),
	)

	return encoder.Encode(manifest)
}

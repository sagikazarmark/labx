package cmd

import (
	"io"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/sagikazarmark/labx/labx"
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
			return runContent(&opts, cmd.OutOrStdout())
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

func runContent(opts *contentOptions, output io.Writer) error {
	fsys, err := os.OpenRoot(opts.path)
	if err != nil {
		return err
	}

	manifest, err := labx.Content(fsys.FS(), opts.channel)
	if err != nil {
		return err
	}

	encoder := yaml.NewEncoder(
		output,
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.IndentSequence(true),
	)

	return encoder.Encode(manifest)
}

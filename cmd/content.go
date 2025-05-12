package cmd

import (
	"os"

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
	root, err := os.OpenRoot(opts.path)
	if err != nil {
		return err
	}

	fsys := root.FS()

	err = labx.Content(fsys, opts.channel)
	if err != nil {
		return err
	}

	return nil
}

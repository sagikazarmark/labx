package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/sagikazarmark/labx/labx"
)

type contentOptions struct {
	path    string
	channel string
	output  string
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

	flags.StringVar(
		&opts.output,
		"output",
		"",
		`Output directory`,
	)

	return cmd
}

func runContent(opts *contentOptions) error {
	root, err := os.OpenRoot(opts.path)
	if err != nil {
		return err
	}

	var outputRoot *os.Root
	if opts.output == "" {
		// Fall back to dist directory within the root
		// Make sure dist exists within root
		err = root.Mkdir("dist", 0755)
		if err != nil && !os.IsExist(err) {
			return err
		}

		outputRoot, err = root.OpenRoot("dist")
		if err != nil {
			return err
		}
	} else {
		// Create the output directory and create an os.Root for it
		err = os.MkdirAll(opts.output, 0755)
		if err != nil {
			return err
		}

		outputRoot, err = os.OpenRoot(opts.output)
		if err != nil {
			return err
		}
	}

	err = labx.Content(root, outputRoot, opts.channel)
	if err != nil {
		return err
	}

	return nil
}

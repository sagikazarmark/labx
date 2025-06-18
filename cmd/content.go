package cmd

import (
	"github.com/spf13/cobra"

	"github.com/sagikazarmark/labx/labx"
)

const defaultOutput = "dist"

type contentOptions struct {
	commonOptions
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

	addCommonFlags(flags, &opts.commonOptions)

	return cmd
}

func runContent(opts *contentOptions) error {
	root, outputRoot, err := setupFsys(&opts.commonOptions)
	if err != nil {
		return err
	}

	err = labx.Content(root, outputRoot, opts.channel)
	if err != nil {
		return err
	}

	return nil
}

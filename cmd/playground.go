package cmd

import (
	"github.com/spf13/cobra"

	"github.com/sagikazarmark/labx/labx"
)

type playgroundOptions struct {
	commonOptions
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

	addCommonFlags(flags, &opts.commonOptions)

	return cmd
}

func runPlayground(opts *playgroundOptions) error {
	root, outputRoot, err := setupFsys(&opts.commonOptions)
	if err != nil {
		return err
	}

	return labx.Playground(root, outputRoot, opts.channel)
}

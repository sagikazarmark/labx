package cmd

import (
	"github.com/spf13/cobra"

	"github.com/sagikazarmark/labx/labx"
)

type generateOptions struct {
	commonOptions
}

func NewGenerateCommand() *cobra.Command {
	var opts generateOptions

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate content based on manifest kind",
		Long: `Generate content based on the kind specified in manifest.yaml.
Automatically routes to appropriate processing:
- playground: generates playground manifest
- other kinds: generates content files`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(&opts)
		},
	}

	flags := cmd.Flags()

	addCommonFlags(flags, &opts.commonOptions)

	return cmd
}

func runGenerate(opts *generateOptions) error {
	root, outputRoot, err := setupFsys(&opts.commonOptions)
	if err != nil {
		return err
	}

	err = labx.Generate(root, outputRoot, opts.channel)
	if err != nil {
		return err
	}

	return nil
}

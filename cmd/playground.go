package cmd

import (
	"github.com/goccy/go-yaml"
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

	manifest, err := labx.Playground(root.FS(), opts.channel)
	if err != nil {
		return err
	}

	if opts.channel == "beta" {
		manifest.Markdown = betaNotice + manifest.Markdown
	}

	// Create the manifest.yaml file
	file, err := outputRoot.Create("manifest.yaml")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(
		file,
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.IndentSequence(true),
	)

	return encoder.Encode(manifest)
}

const betaNotice = `::remark-box
---
kind: warning
---

⚠️ This content is marked as **beta**, meaning it's unfinished or still in progress and may change significantly.
::

`

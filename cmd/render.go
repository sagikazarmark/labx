package cmd

import (
	"io/fs"
	"os"

	"github.com/spf13/cobra"

	"github.com/sagikazarmark/labx/labx"
)

type renderOptions struct {
	path         string
	output       string
	channel      string
	templateDirs []string
	dataDirs     []string
}

func NewRenderCommand() *cobra.Command {
	var opts renderOptions

	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render markdown templates without manifest transforms",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRender(&opts)
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
		&opts.output,
		"output",
		"",
		`Output directory`,
	)

	flags.StringVar(
		&opts.channel,
		"channel",
		"dev",
		`Which channel to use`,
	)

	flags.StringSliceVar(
		&opts.templateDirs,
		"template-dir",
		[]string{},
		`Global template directories to load .md files from (loaded before content templates, can be specified multiple times)`,
	)

	flags.StringSliceVar(
		&opts.dataDirs,
		"data-dir",
		[]string{},
		`Additional data directories to load JSON files from (can be specified multiple times)`,
	)

	_ = cmd.MarkFlagRequired("output")

	return cmd
}

func runRender(opts *renderOptions) error {
	root, outputRoot, err := setupRenderFsys(opts)
	if err != nil {
		return err
	}

	var templateFSs []fs.FS
	for _, templateDir := range opts.templateDirs {
		templateFSs = append(templateFSs, os.DirFS(templateDir))
	}

	var dataFSs []fs.FS
	for _, dataDir := range opts.dataDirs {
		dataFSs = append(dataFSs, os.DirFS(dataDir))
	}

	return labx.Render(labx.GenerateOpts{
		Root:         root,
		Output:       outputRoot,
		Channel:      opts.channel,
		TemplateDirs: templateFSs,
		DataDirs:     dataFSs,
	})
}

func setupRenderFsys(opts *renderOptions) (*os.Root, *os.Root, error) {
	root, err := os.OpenRoot(opts.path)
	if err != nil {
		return nil, nil, err
	}

	err = os.MkdirAll(opts.output, 0o755)
	if err != nil {
		return nil, nil, err
	}

	outputRoot, err := os.OpenRoot(opts.output)
	if err != nil {
		return nil, nil, err
	}

	return root, outputRoot, nil
}

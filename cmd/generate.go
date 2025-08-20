package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/sagikazarmark/labx/labx"
)

const defaultOutput = "dist"

type generateOptions struct {
	path         string
	output       string
	clear        bool
	channel      string
	templateDirs []string
	dataDirs     []string
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

	addFlags(flags, &opts)

	return cmd
}

// addFlags adds the flags to the generate command
func addFlags(flags *pflag.FlagSet, opts *generateOptions) {
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

	flags.BoolVar(
		&opts.clear,
		"clear",
		false,
		`Clear output directory before generating content`,
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
}

func runGenerate(opts *generateOptions) error {
	root, outputRoot, err := setupFsys(opts)
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

	generateOpts := labx.GenerateOpts{
		Root:         root,
		Output:       outputRoot,
		Channel:      opts.channel,
		TemplateDirs: templateFSs,
		DataDirs:     dataFSs,
	}

	err = labx.Generate(generateOpts)
	if err != nil {
		return err
	}

	return nil
}

// setupFsys handles the output directory setup logic
func setupFsys(opts *generateOptions) (*os.Root, *os.Root, error) {
	root, err := os.OpenRoot(opts.path)
	if err != nil {
		return nil, nil, err
	}

	var outputPath string
	if opts.output == "" {
		outputPath = filepath.Join(opts.path, defaultOutput)
	} else {
		outputPath = opts.output
	}

	// If clear is true, always remove the directory first
	if opts.clear {
		err = os.RemoveAll(outputPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, nil, err
		}
	}

	// Create the output directory
	err = os.MkdirAll(outputPath, 0o755)
	if err != nil {
		return nil, nil, err
	}

	// If clear is false, check if directory is empty
	if !opts.clear {
		if dirExists, err := isDirEmptyPath(outputPath); err != nil {
			return nil, nil, err
		} else if !dirExists {
			return nil, nil, fmt.Errorf("output directory '%s' is not empty. Use --clear to remove it first", outputPath)
		}
	}

	// Create the os.Root instance for output
	outputRoot, err := os.OpenRoot(outputPath)
	if err != nil {
		return nil, nil, err
	}

	return root, outputRoot, nil
}

// isDirEmptyPath checks if a directory path is empty or doesn't exist
// Returns true if directory is empty or doesn't exist, false if it contains files
func isDirEmptyPath(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/sagikazarmark/labx/labx"
)

const defaultOutput = "dist"

type generateOptions struct {
	path    string
	output  string
	clear   bool
	channel string
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
}

func runGenerate(opts *generateOptions) error {
	root, outputRoot, err := setupFsys(opts)
	if err != nil {
		return err
	}

	err = labx.Generate(root, outputRoot, opts.channel)
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

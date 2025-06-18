package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/sagikazarmark/labx/labx"
)

const defaultOutput = "dist"

type contentOptions struct {
	path    string
	channel string
	output  string
	clear   bool
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

	flags.BoolVar(
		&opts.clear,
		"clear",
		false,
		`Clear output directory before generating content`,
	)

	return cmd
}

func runContent(opts *contentOptions) error {
	root, err := os.OpenRoot(opts.path)
	if err != nil {
		return err
	}

	var outputRoot *os.Root
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
			return err
		}
	}

	// Create the output directory
	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		return err
	}

	// Create the os.Root instance
	outputRoot, err = os.OpenRoot(outputPath)
	if err != nil {
		return err
	}

	// If clear is false, check if directory is empty
	if !opts.clear {
		if dirExists, err := isDirEmptyPath(outputPath); err != nil {
			return err
		} else if !dirExists {
			return fmt.Errorf("output directory '%s' is not empty. Use --clear to remove it first", outputPath)
		}
	}

	err = labx.Content(root, outputRoot, opts.channel)
	if err != nil {
		return err
	}

	return nil
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

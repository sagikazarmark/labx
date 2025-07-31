package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	xcmd "github.com/sagikazarmark/labx/cmd"
	"github.com/sagikazarmark/labx/internal/config"
)

func main() {
	cli := xcmd.NewCli()

	cmd := &cobra.Command{
		Use:     "labx <command>",
		Short:   "labx - opinionated tools for iximiuz Labs content",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			cli.Init(cfg, version)

			return nil
		},
	}

	cmd.AddCommand(
		xcmd.NewGenerateCommand(),
		xcmd.NewCheckTasksCommand(cli),
	)

	err := cmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func loadConfig() (*config.Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("Unable to determine home directory: %w", err)
	}

	cfg, err := config.Load(homeDir)
	if err != nil {
		return nil, fmt.Errorf("Unable to load config: %w", err)
	}

	return cfg, nil
}

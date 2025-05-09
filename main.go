package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	xcmd "github.com/sagikazarmark/labx/cmd"
)

func main() {
	cmd := &cobra.Command{
		Use:     "labx <playground|content>",
		Short:   "labx - opinionated tools for iximiuz Labs content",
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
		},
	}

	cmd.AddCommand(
		xcmd.NewPlaygroundCommand(),
		xcmd.NewContentCommand(),
	)

	err := cmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/iximiuz/labctl/api"
	"github.com/spf13/cobra"
)

type checkTasksOptions struct {
	timeout time.Duration
}

func NewCheckTasksCommand(cli *Cli) *cobra.Command {
	var opts checkTasksOptions

	cmd := &cobra.Command{
		Use:   "check-tasks <playId>",
		Short: "Check if all tasks of a play are successful",
		Long:  `Check if all tasks of the given play are successful. Returns success if all tasks are completed.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cli.client == nil {
				return fmt.Errorf("API client not initialized")
			}

			playID := args[0]

			return runCheckTasks(cmd.Context(), cli.client, playID, &opts)
		},
	}

	flags := cmd.Flags()
	flags.DurationVar(
		&opts.timeout,
		"timeout",
		5*time.Minute,
		"Timeout for checking tasks",
	)

	return cmd

}

func runCheckTasks(ctx context.Context, client *api.Client, playID string, opts *checkTasksOptions) error {
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()

	// Polling interval
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout reached while waiting for tasks to complete")
		case <-ticker.C:
			play, err := client.GetPlay(timeoutCtx, playID)
			if err != nil {
				return fmt.Errorf("failed to get play %s: %w", playID, err)
			}

			// Check if all tasks are successful
			allSuccessful := true
			failedTasks := []string{}
			pendingTasks := []string{}

			for name, task := range play.Tasks {
				switch task.Status {
				case api.PlayTaskStatusCompleted:
					// Task is successful, continue
				case api.PlayTaskStatusFailed:
					allSuccessful = false
					failedTasks = append(failedTasks, name)
				default:
					allSuccessful = false
					pendingTasks = append(pendingTasks, name)
				}
			}

			if len(failedTasks) > 0 {
				return fmt.Errorf("tasks failed: %v", failedTasks)
			}

			if allSuccessful {
				fmt.Printf("All tasks for play %s are successful\n", playID)
				return nil
			}

			fmt.Printf("Waiting for tasks to complete. Pending: %v\n", pendingTasks)
		}
	}
}

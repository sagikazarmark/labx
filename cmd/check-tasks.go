package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
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

	// Configure exponential backoff
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 5 * time.Second
	b.MaxInterval = 30 * time.Second

	operation := func() (bool, error) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		play, err := client.GetPlay(ctx, playID)
		if err != nil {
			return false, getPlayError(playID, err)
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

		// Failed tasks are a permanent error
		if len(failedTasks) > 0 {
			return false, backoff.Permanent(fmt.Errorf("tasks failed: %v", failedTasks))
		}

		if allSuccessful {
			fmt.Printf("All tasks for play %s are successful\n", playID)

			return true, nil
		}

		fmt.Printf("Waiting for tasks to complete. Pending: %v\n", pendingTasks)

		return false, fmt.Errorf("tasks still pending: %v", pendingTasks)
	}

	_, err := backoff.Retry(
		timeoutCtx,
		operation,
		backoff.WithBackOff(b),
		backoff.WithMaxElapsedTime(opts.timeout),
	)

	return err
}

func getPlayError(playID string, err error) error {
	if err == nil {
		return nil
	}

	nerr := fmt.Errorf("failed to get play %s: %w", playID, err)

	// Check if error is api.ErrNotFound
	if errors.Is(err, api.ErrNotFound) {
		return backoff.Permanent(nerr)
	}

	// Check if error message starts with "request failed with status 4" (client errors)
	errMsg := err.Error()
	if strings.HasPrefix(errMsg, "request failed with status 4") {
		return backoff.Permanent(nerr)
	}

	return nerr
}

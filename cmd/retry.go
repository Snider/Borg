package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Snider/Borg/pkg/failures"
	"github.com/spf13/cobra"
)

var retryCmd = &cobra.Command{
	Use:   "retry [run-directory]",
	Short: "Retry failures from a collection run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Retrying failures from %s...\n", args[0])

		onlyRetryable, _ := cmd.Flags().GetBool("only-retryable")

		reportPath := filepath.Join(args[0], "failures.json")
		data, err := os.ReadFile(reportPath)
		if err != nil {
			return fmt.Errorf("failed to read failure report: %w", err)
		}

		var report failures.FailureReport
		if err := json.Unmarshal(data, &report); err != nil {
			return fmt.Errorf("failed to parse failure report: %w", err)
		}

		for _, failure := range report.Failures {
			if onlyRetryable && !failure.Retryable {
				fmt.Printf("Skipping non-retryable failure: %s\n", failure.URL)
				continue
			}

			fmt.Printf("Retrying %s...\n", failure.URL)
			retryCmd := exec.Command("borg", "collect", "github", "repo", failure.URL)
			retryCmd.Stdout = os.Stdout
			retryCmd.Stderr = os.Stderr
			if err := retryCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to retry %s: %v\n", failure.URL, err)
			}
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(retryCmd)
	retryCmd.Flags().Bool("only-retryable", false, "Retry only failures marked as retryable")
}

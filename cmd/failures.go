package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Snider/Borg/pkg/failures"
	"github.com/spf13/cobra"
)

var failuresCmd = &cobra.Command{
	Use:   "failures",
	Short: "Manage failures from collection runs",
}

var failuresShowCmd = &cobra.Command{
	Use:   "show [run-directory]",
	Short: "Show a summary of a failure report",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reportPath := filepath.Join(args[0], "failures.json")
		data, err := os.ReadFile(reportPath)
		if err != nil {
			return fmt.Errorf("failed to read failure report: %w", err)
		}

		var report failures.FailureReport
		if err := json.Unmarshal(data, &report); err != nil {
			return fmt.Errorf("failed to parse failure report: %w", err)
		}

		fmt.Printf("Collection: %s\n", report.Collection)
		fmt.Printf("Started:    %s\n", report.Started.Format(time.RFC3339))
		fmt.Printf("Completed:  %s\n", report.Completed.Format(time.RFC3339))
		fmt.Printf("Total:      %d\n", report.Stats.Total)
		fmt.Printf("Success:    %d\n", report.Stats.Success)
		fmt.Printf("Failed:     %d\n", report.Stats.Failed)

		if len(report.Failures) > 0 {
			fmt.Println("\nFailures:")
			for _, f := range report.Failures {
				fmt.Printf("  - URL: %s\n", f.URL)
				fmt.Printf("    Error: %s\n", f.Error)
			}
		}

		return nil
	},
}

var failuresClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear old failure reports",
	RunE: func(cmd *cobra.Command, args []string) error {
		olderThan, _ := cmd.Flags().GetString("older-than")
		failuresDir, _ := cmd.Flags().GetString("failures-dir")
		if failuresDir == "" {
			failuresDir = ".borg-failures"
		}

		duration, err := time.ParseDuration(olderThan)
		if err != nil {
			return fmt.Errorf("invalid duration for --older-than: %w", err)
		}

		cutoff := time.Now().Add(-duration)

		entries, err := os.ReadDir(failuresDir)
		if err != nil {
			return fmt.Errorf("failed to read failures directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				runTime, err := time.Parse("2006-01-02T15-04-05", entry.Name())
				if err != nil {
					// Ignore directories that don't match the timestamp format
					continue
				}

				if runTime.Before(cutoff) {
					runPath := filepath.Join(failuresDir, entry.Name())
					fmt.Printf("Removing old failure directory: %s\n", runPath)
					if err := os.RemoveAll(runPath); err != nil {
						fmt.Fprintf(os.Stderr, "failed to remove %s: %v\n", runPath, err)
					}
				}
			}
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(failuresCmd)
	failuresCmd.AddCommand(failuresShowCmd)
	failuresCmd.AddCommand(failuresClearCmd)

	failuresClearCmd.Flags().String("older-than", "720h", "Clear failures older than this duration (e.g., 7d, 24h)")
	failuresClearCmd.Flags().String("failures-dir", ".borg-failures", "The directory where failures are stored")
}

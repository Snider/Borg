package cmd

import (
	"context"
	"log/slog"

	"github.com/Snider/Borg/pkg/retry"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "borg",
		Short: "A tool for collecting and managing data.",
		Long: `Borg Data Collector is a command-line tool for cloning Git repositories,
packaging their contents into a single file, and managing the data within.`,
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().Int("retries", 3, "Max retry attempts")
	rootCmd.PersistentFlags().Duration("retry-backoff", 1s, "Initial backoff duration")
	rootCmd.PersistentFlags().Duration("retry-max", 30s, "Maximum backoff duration")
	rootCmd.PersistentFlags().Float64("retry-jitter", 0.1, "Randomization factor")
	rootCmd.PersistentFlags().Bool("no-retry", false, "Disable retries")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Configure retry settings
		retries, _ := cmd.Flags().GetInt("retries")
		retryBackoff, _ := cmd.Flags().GetDuration("retry-backoff")
		retryMax, _ := cmd.Flags().GetDuration("retry-max")
		retryJitter, _ := cmd.Flags().GetFloat64("retry-jitter")
		noRetry, _ := cmd.Flags().GetBool("no-retry")

		if noRetry {
			retry.DefaultTransport.Retries = 0
		} else {
			retry.DefaultTransport.Retries = retries
			retry.DefaultTransport.InitialBackoff = retryBackoff
			retry.DefaultTransport.MaxBackoff = retryMax
			retry.DefaultTransport.Jitter = retryJitter
		}
	}

	return rootCmd
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = NewRootCmd()

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(log *slog.Logger) error {
	RootCmd.SetContext(context.WithValue(context.Background(), "logger", log))
	return RootCmd.Execute()
}

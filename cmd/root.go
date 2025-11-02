package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "borg-data-collector",
	Short: "A tool for collecting and managing data.",
	Long: `Borg Data Collector is a command-line tool for cloning Git repositories,
packaging their contents into a single file, and managing the data within.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(log *slog.Logger) error {
	RootCmd.SetContext(context.WithValue(context.Background(), "logger", log))
	return RootCmd.Execute()
}

// init configures persistent flags for the root command.
func init() {
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")
}

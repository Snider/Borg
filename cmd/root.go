package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "borg-data-collector",
		Short: "A tool for collecting and managing data.",
		Long: `Borg Data Collector is a command-line tool for cloning Git repositories,
packaging their contents into a single file, and managing the data within.`,
	}
	rootCmd.AddCommand(allCmd)
	rootCmd.AddCommand(collectCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")
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

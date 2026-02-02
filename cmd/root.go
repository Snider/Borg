package cmd

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Snider/Borg/pkg/templates"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "borg",
		Short: "A tool for collecting and managing data.",
		Long: `Borg Data Collector is a command-line tool for cloning Git repositories,
packaging their contents into a single file, and managing the data within.`,
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// Don't log template or help commands
			if cmd.Parent().Name() == "template" || cmd.Name() == "help" {
				return nil
			}
			if err := templates.AppendToHistory(cmd.CommandPath() + " " + strings.Join(args, " ")); err != nil {
				log := cmd.Context().Value("logger").(*slog.Logger)
				log.Warn("could not write to history", "error", err)
			}
			return nil
		},
	}

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

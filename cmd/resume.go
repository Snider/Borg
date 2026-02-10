package cmd

import (
	"fmt"
	"strings"

	"github.com/Snider/Borg/pkg/progress"
	"github.com/spf13/cobra"
)

func NewResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume [.borg-progress-file]",
		Short: "Resume an interrupted collection from a progress file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			progressFile := args[0]
			p, err := progress.Load(progressFile)
			if err != nil {
				return fmt.Errorf("failed to load progress file: %w", err)
			}

			parts := strings.Split(p.Source, ":")
			if len(parts) < 3 {
				return fmt.Errorf("invalid source format in progress file: %s", p.Source)
			}

			// Reconstruct and execute the original command with --resume
			originalCmd := []string{"collect"}
			originalCmd = append(originalCmd, strings.Split(parts[0], "/")...)
			originalCmd = append(originalCmd, parts[1])
			originalCmd = append(originalCmd, parts[2])
			originalCmd = append(originalCmd, "--resume")

			rootCmd := cmd.Root()
			rootCmd.SetArgs(originalCmd)

			fmt.Fprintf(cmd.OutOrStdout(), "Resuming with command: %s\n", strings.Join(originalCmd, " "))
			return rootCmd.Execute()
		},
	}
}

func init() {
	RootCmd.AddCommand(NewResumeCmd())
}

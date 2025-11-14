package cmd

import (
	"github.com/Snider/Borg/pkg/matrix"
	"github.com/spf13/cobra"
)

var runCmd = NewRunCmd()

func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [matrix file]",
		Short: "Run a Terminal Isolation Matrix.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return matrix.Run(args[0])
		},
	}
}

func GetRunCmd() *cobra.Command {
	return runCmd
}

func init() {
	RootCmd.AddCommand(GetRunCmd())
}

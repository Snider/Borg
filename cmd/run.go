package cmd

import (
	"github.com/Snider/Borg/pkg/tim"
	"github.com/spf13/cobra"
)

var runCmd = NewRunCmd()

func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [tim file]",
		Short: "Run a Terminal Isolation Matrix.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return tim.Run(args[0])
		},
	}
}

func GetRunCmd() *cobra.Command {
	return runCmd
}

func init() {
	RootCmd.AddCommand(GetRunCmd())
}

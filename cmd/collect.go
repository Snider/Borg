package cmd

import (
	"github.com/spf13/cobra"
)

// collectCmd represents the collect command
var collectCmd = NewCollectCmd()

func init() {
	RootCmd.AddCommand(GetCollectCmd())
}
func NewCollectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect a resource from a URI.",
		Long:  `Collect a resource from a URI and store it in a DataNode.`,
	}
	cmd.PersistentFlags().String("on-failure", "continue", "Action to take on failure: continue, stop, prompt")
	cmd.PersistentFlags().String("failures-dir", ".borg-failures", "Directory to store failure reports")
	return cmd
}

func GetCollectCmd() *cobra.Command {
	return collectCmd
}

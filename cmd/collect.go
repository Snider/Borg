package cmd

import (
	"github.com/spf13/cobra"
)

// collectCmd represents the collect command
var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect a resource from a URI.",
	Long:  `Collect a resource from a URI and store it in a DataNode.`,
}

func init() {
	RootCmd.AddCommand(collectCmd)
}
func NewCollectCmd() *cobra.Command {
	return collectCmd
}

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

// init registers the collect command with the root.
func init() {
	RootCmd.AddCommand(collectCmd)
}
func NewCollectCmd() *cobra.Command {
	return collectCmd
}

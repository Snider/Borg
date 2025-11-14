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
	return &cobra.Command{
		Use:   "collect",
		Short: "Collect a resource from a URI.",
		Long:  `Collect a resource from a URI and store it in a DataNode.`,
	}
}

func GetCollectCmd() *cobra.Command {
	return collectCmd
}

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
	cmd.PersistentFlags().String("bandwidth", "0", "Limit download bandwidth (e.g., 1MB/s, 500KB/s, 0 for unlimited)")
	return cmd
}

func GetCollectCmd() *cobra.Command {
	return collectCmd
}

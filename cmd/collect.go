package cmd

import (
	"github.com/spf13/cobra"
)

// collectCmd represents the collect command
var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect a resource and store it in a DataNode.",
	Long:  `Collect a resource from a git repository, a website, or other URI and store it in a DataNode.`,
}

// init registers the 'collect' command under the root command.
func init() {
	RootCmd.AddCommand(collectCmd)
}

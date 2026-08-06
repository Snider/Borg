package cmd

import (
	"github.com/spf13/cobra"
)

// collectArchiveCmd represents the collect archive command
var collectArchiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Collect a resource from the Internet Archive.",
	Long:  `Collect a resource from the Internet Archive, such as a search query, an item, or a collection.`,
}

func init() {
	collectCmd.AddCommand(collectArchiveCmd)
}

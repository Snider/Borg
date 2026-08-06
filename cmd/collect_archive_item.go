package cmd

import (
	"github.com/Snider/Borg/pkg/archive"
	"github.com/spf13/cobra"
)

// collectArchiveItemCmd represents the collect archive item command
var collectArchiveItemCmd = &cobra.Command{
	Use:   "item [identifier]",
	Short: "Collect an item from the Internet Archive.",
	Long:  `Collect an item and all of its files from the Internet Archive.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]
		return archive.DownloadItem(identifier, "archive", "")
	},
}

func init() {
	collectArchiveCmd.AddCommand(collectArchiveItemCmd)
}

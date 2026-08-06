package cmd

import (
	"fmt"
	"github.com/Snider/Borg/pkg/archive"
	"github.com/spf13/cobra"
)

// collectArchiveSearchCmd represents the collect archive search command
var collectArchiveSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for items on the Internet Archive.",
	Long:  `Search for items on the Internet Archive and collect them.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		mediaType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt("limit")
		format, _ := cmd.Flags().GetString("format")

		items, err := archive.Search(query, mediaType, limit)
		if err != nil {
			return err
		}

		for _, item := range items {
			if err := archive.DownloadItem(item.Identifier, "archive", format); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error downloading item %s: %v\n", item.Identifier, err)
			}
		}

		return nil
	},
}

func init() {
	collectArchiveCmd.AddCommand(collectArchiveSearchCmd)
	collectArchiveSearchCmd.Flags().String("type", "", "Filter by mediatype (texts, software)")
	collectArchiveSearchCmd.Flags().Int("limit", 10, "Max items to collect")
	collectArchiveSearchCmd.Flags().String("format", "", "Preferred file format")
}

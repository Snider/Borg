package cmd

import (
	"fmt"
	"github.com/Snider/Borg/pkg/archive"
	"github.com/spf13/cobra"
)

// collectArchiveCollectionCmd represents the collect archive collection command
var collectArchiveCollectionCmd = &cobra.Command{
	Use:   "collection [identifier]",
	Short: "Collect a collection from the Internet Archive.",
	Long:  `Collect a collection and all of its items from the Internet Archive.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]
		items, err := archive.GetCollection(identifier)
		if err != nil {
			return err
		}

		for _, item := range items {
			if err := archive.DownloadItem(item.Identifier, "archive", ""); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error downloading item %s from collection: %v\n", item.Identifier, err)
			}
		}

		return nil
	},
}

func init() {
	collectArchiveCmd.AddCommand(collectArchiveCollectionCmd)
}

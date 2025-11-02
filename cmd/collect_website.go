package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/matrix"
	"github.com/Snider/Borg/pkg/ui"
	"github.com/Snider/Borg/pkg/website"

	"github.com/spf13/cobra"
)

// collectWebsiteCmd represents the collect website command
var collectWebsiteCmd = &cobra.Command{
	Use:   "website [url]",
	Short: "Collect a single website",
	Long:  `Collect a single website and store it in a DataNode.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		websiteURL := args[0]
		outputFile, _ := cmd.Flags().GetString("output")
		depth, _ := cmd.Flags().GetInt("depth")
		format, _ := cmd.Flags().GetString("format")

		bar := ui.NewProgressBar(-1, "Crawling website")
		defer bar.Finish()

		dn, err := website.DownloadAndPackageWebsite(websiteURL, depth, bar)
		if err != nil {
			fmt.Printf("Error downloading and packaging website: %v\n", err)
			return
		}

		var data []byte
		if format == "matrix" {
			matrix, err := matrix.FromDataNode(dn)
			if err != nil {
				fmt.Printf("Error creating matrix: %v\n", err)
				return
			}
			data, err = matrix.ToTar()
			if err != nil {
				fmt.Printf("Error serializing matrix: %v\n", err)
				return
			}
		} else {
			data, err = dn.ToTar()
			if err != nil {
				fmt.Printf("Error serializing DataNode: %v\n", err)
				return
			}
		}

		err = os.WriteFile(outputFile, data, 0644)
		if err != nil {
			fmt.Printf("Error writing website to file: %v\n", err)
			return
		}

		fmt.Printf("Website saved to %s\n", outputFile)
	},
}

func init() {
	collectCmd.AddCommand(collectWebsiteCmd)
	collectWebsiteCmd.PersistentFlags().String("output", "website.dat", "Output file for the DataNode")
	collectWebsiteCmd.PersistentFlags().Int("depth", 2, "Recursion depth for downloading")
	collectWebsiteCmd.PersistentFlags().String("format", "datanode", "Output format (datanode or matrix)")
}

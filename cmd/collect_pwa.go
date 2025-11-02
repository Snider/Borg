package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/matrix"
	"github.com/Snider/Borg/pkg/pwa"
	"github.com/Snider/Borg/pkg/ui"

	"github.com/spf13/cobra"
)

// collectPWACmd represents the collect pwa command
var collectPWACmd = &cobra.Command{
	Use:   "pwa",
	Short: "Collect a single PWA using a URI",
	Long: `Collect a single PWA and store it in a DataNode.

Example:
  borg collect pwa --uri https://example.com --output mypwa.dat`,
	Run: func(cmd *cobra.Command, args []string) {
		pwaURL, _ := cmd.Flags().GetString("uri")
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")

		if pwaURL == "" {
			fmt.Println("Error: uri is required")
			return
		}

		bar := ui.NewProgressBar(-1, "Finding PWA manifest")
		defer bar.Finish()

		manifestURL, err := pwa.FindManifest(pwaURL)
		if err != nil {
			fmt.Printf("Error finding manifest: %v\n", err)
			return
		}
		bar.Describe("Downloading and packaging PWA")
		dn, err := pwa.DownloadAndPackagePWA(pwaURL, manifestURL, bar)
		if err != nil {
			fmt.Printf("Error downloading and packaging PWA: %v\n", err)
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
			fmt.Printf("Error writing PWA to file: %v\n", err)
			return
		}

		fmt.Printf("PWA saved to %s\n", outputFile)
	},
}

func init() {
	collectCmd.AddCommand(collectPWACmd)
	collectPWACmd.Flags().String("uri", "", "The URI of the PWA to collect")
	collectPWACmd.Flags().String("output", "pwa.dat", "Output file for the DataNode")
	collectPWACmd.Flags().String("format", "datanode", "Output format (datanode or matrix)")
}

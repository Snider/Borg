package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/pwa"

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

		if pwaURL == "" {
			fmt.Println("Error: uri is required")
			return
		}

		fmt.Println("Finding PWA manifest...")
		manifestURL, err := pwa.FindManifest(pwaURL)
		if err != nil {
			fmt.Printf("Error finding manifest: %v\n", err)
			return
		}
		fmt.Printf("Found manifest: %s\n", manifestURL)

		fmt.Println("Downloading and packaging PWA...")
		dn, err := pwa.DownloadAndPackagePWA(pwaURL, manifestURL)
		if err != nil {
			fmt.Printf("Error downloading and packaging PWA: %v\n", err)
			return
		}

		pwaData, err := dn.ToTar()
		if err != nil {
			fmt.Printf("Error converting PWA to bytes: %v\n", err)
			return
		}

		err = os.WriteFile(outputFile, pwaData, 0644)
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
}

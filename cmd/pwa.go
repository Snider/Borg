package cmd

import (
	"fmt"
	"os"

	"borg-data-collector/pkg/pwa"

	"github.com/spf13/cobra"
)

// pwaCmd represents the pwa command
var pwaCmd = &cobra.Command{
	Use:   "pwa [url]",
	Short: "Download a PWA from a URL",
	Long:  `Downloads a Progressive Web Application (PWA) from a given URL by finding its manifest.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pwaURL := args[0]
		outputFile, _ := cmd.Flags().GetString("output")

		fmt.Println("Finding PWA manifest...")
		manifestURL, err := pwa.FindManifestURL(pwaURL)
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
			fmt.Printf("Error serializing PWA data: %v\n", err)
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
	rootCmd.AddCommand(pwaCmd)
	pwaCmd.PersistentFlags().String("output", "pwa.dat", "Output file for the PWA DataNode")
}

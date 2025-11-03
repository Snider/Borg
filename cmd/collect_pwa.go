package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/matrix"
	"github.com/Snider/Borg/pkg/pwa"
	"github.com/Snider/Borg/pkg/ui"

	"github.com/spf13/cobra"
)

var (
	// PWAClient is the pwa client used by the command. It can be replaced for testing.
	PWAClient = pwa.NewPWAClient()
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
		compression, _ := cmd.Flags().GetString("compression")

		if pwaURL == "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "Error: uri is required")
			return
		}

		bar := ui.NewProgressBar(-1, "Finding PWA manifest")
		defer bar.Finish()

		manifestURL, err := PWAClient.FindManifest(pwaURL)
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Error finding manifest:", err)
			return
		}
		bar.Describe("Downloading and packaging PWA")
		dn, err := PWAClient.DownloadAndPackagePWA(pwaURL, manifestURL, bar)
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Error downloading and packaging PWA:", err)
			return
		}

		var data []byte
		if format == "matrix" {
			matrix, err := matrix.FromDataNode(dn)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "Error creating matrix:", err)
				return
			}
			data, err = matrix.ToTar()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "Error serializing matrix:", err)
				return
			}
		} else {
			data, err = dn.ToTar()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "Error serializing DataNode:", err)
				return
			}
		}

		compressedData, err := compress.Compress(data, compression)
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Error compressing data:", err)
			return
		}

		if outputFile == "" {
			outputFile = "pwa." + format
			if compression != "none" {
				outputFile += "." + compression
			}
		}

		err = os.WriteFile(outputFile, compressedData, 0644)
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Error writing PWA to file:", err)
			return
		}

		fmt.Fprintln(cmd.OutOrStdout(), "PWA saved to", outputFile)
	},
}

// init registers the 'collect pwa' subcommand and its flags.
func init() {
	collectCmd.AddCommand(collectPWACmd)
	collectPWACmd.Flags().String("uri", "", "The URI of the PWA to collect")
	collectPWACmd.Flags().String("output", "", "Output file for the DataNode")
	collectPWACmd.Flags().String("format", "datanode", "Output format (datanode or matrix)")
	collectPWACmd.Flags().String("compression", "none", "Compression format (none, gz, or xz)")
}

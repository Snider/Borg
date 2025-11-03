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

type CollectPWACmd struct {
	cobra.Command
	PWAClient pwa.PWAClient
}

// NewCollectPWACmd creates a new collect pwa command
func NewCollectPWACmd() *CollectPWACmd {
	c := &CollectPWACmd{
		PWAClient: pwa.NewPWAClient(),
	}
	c.Command = cobra.Command{
		Use:   "pwa",
		Short: "Collect a single PWA using a URI",
		Long: `Collect a single PWA and store it in a DataNode.

Example:
  borg collect pwa --uri https://example.com --output mypwa.dat`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pwaURL, _ := cmd.Flags().GetString("uri")
			outputFile, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")

			finalPath, err := CollectPWA(c.PWAClient, pwaURL, outputFile, format, compression)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "PWA saved to", finalPath)
			return nil
		},
	}
	c.Flags().String("uri", "", "The URI of the PWA to collect")
	c.Flags().String("output", "", "Output file for the DataNode")
	c.Flags().String("format", "datanode", "Output format (datanode or matrix)")
	c.Flags().String("compression", "none", "Compression format (none, gz, or xz)")
	return c
}

func init() {
	collectCmd.AddCommand(&NewCollectPWACmd().Command)
}
func CollectPWA(client pwa.PWAClient, pwaURL string, outputFile string, format string, compression string) (string, error) {
	if pwaURL == "" {
		return "", fmt.Errorf("uri is required")
	}
	if format != "datanode" && format != "matrix" {
		return "", fmt.Errorf("invalid format: %s (must be 'datanode' or 'matrix')", format)
	}
	if compression != "none" && compression != "gz" && compression != "xz" {
		return "", fmt.Errorf("invalid compression: %s (must be 'none', 'gz', or 'xz')", compression)
	}

	bar := ui.NewProgressBar(-1, "Finding PWA manifest")
	defer bar.Finish()

	manifestURL, err := client.FindManifest(pwaURL)
	if err != nil {
		return "", fmt.Errorf("error finding manifest: %w", err)
	}
	bar.Describe("Downloading and packaging PWA")
	dn, err := client.DownloadAndPackagePWA(pwaURL, manifestURL, bar)
	if err != nil {
		return "", fmt.Errorf("error downloading and packaging PWA: %w", err)
	}

	var data []byte
	if format == "matrix" {
		matrix, err := matrix.FromDataNode(dn)
		if err != nil {
			return "", fmt.Errorf("error creating matrix: %w", err)
		}
		data, err = matrix.ToTar()
		if err != nil {
			return "", fmt.Errorf("error serializing matrix: %w", err)
		}
	} else {
		data, err = dn.ToTar()
		if err != nil {
			return "", fmt.Errorf("error serializing DataNode: %w", err)
		}
	}

	compressedData, err := compress.Compress(data, compression)
	if err != nil {
		return "", fmt.Errorf("error compressing data: %w", err)
	}

	if outputFile == "" {
		outputFile = "pwa." + format
		if compression != "none" {
			outputFile += "." + compression
		}
	}

	err = os.WriteFile(outputFile, compressedData, 0644)
	if err != nil {
		return "", fmt.Errorf("error writing PWA to file: %w", err)
	}
	return outputFile, nil
}

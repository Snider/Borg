package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/httpclient"
	"github.com/Snider/Borg/pkg/pwa"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/Snider/Borg/pkg/ui"

	"github.com/spf13/cobra"
)

// NewCollectPWACmd creates a new collect pwa command
func NewCollectPWACmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pwa [url]",
		Short: "Collect a single PWA using a URI",
		Long: `Collect a single PWA and store it in a DataNode.

Examples:
  borg collect pwa https://example.com
  borg collect pwa https://example.com --output mypwa.dat
  borg collect pwa https://example.com --format stim --password secret`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pwaURL, _ := cmd.Flags().GetString("uri")
			// Allow URL as positional argument
			if len(args) > 0 && pwaURL == "" {
				pwaURL = args[0]
			}
			outputFile, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")
			password, _ := cmd.Flags().GetString("password")

			totalTimeout, _ := cmd.Flags().GetDuration("timeout")
			connectTimeout, _ := cmd.Flags().GetDuration("connect-timeout")
			tlsTimeout, _ := cmd.Flags().GetDuration("tls-timeout")
			headerTimeout, _ := cmd.Flags().GetDuration("header-timeout")

			httpClient := httpclient.NewClient(totalTimeout, connectTimeout, tlsTimeout, headerTimeout)
			pwaClient := pwa.NewPWAClient(httpClient)

			finalPath, err := CollectPWA(pwaClient, pwaURL, outputFile, format, compression, password)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "PWA saved to", finalPath)
			return nil
		},
	}
	cmd.Flags().String("uri", "", "The URI of the PWA to collect (can also be passed as positional arg)")
	cmd.Flags().String("output", "", "Output file for the DataNode")
	cmd.Flags().String("format", "datanode", "Output format (datanode, tim, trix, or stim)")
	cmd.Flags().String("compression", "none", "Compression format (none, gz, or xz)")
	cmd.Flags().String("password", "", "Password for encryption (required for stim format)")
	return cmd
}

func init() {
	collectCmd.AddCommand(NewCollectPWACmd())
}
func CollectPWA(client pwa.PWAClient, pwaURL string, outputFile string, format string, compression string, password string) (string, error) {
	if pwaURL == "" {
		return "", fmt.Errorf("url is required")
	}
	if format != "datanode" && format != "tim" && format != "trix" && format != "stim" {
		return "", fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', 'trix', or 'stim')", format)
	}
	if format == "stim" && password == "" {
		return "", fmt.Errorf("password is required for stim format")
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
	if format == "tim" {
		t, err := tim.FromDataNode(dn)
		if err != nil {
			return "", fmt.Errorf("error creating tim: %w", err)
		}
		data, err = t.ToTar()
		if err != nil {
			return "", fmt.Errorf("error serializing tim: %w", err)
		}
	} else if format == "stim" {
		t, err := tim.FromDataNode(dn)
		if err != nil {
			return "", fmt.Errorf("error creating tim: %w", err)
		}
		data, err = t.ToSigil(password)
		if err != nil {
			return "", fmt.Errorf("error encrypting stim: %w", err)
		}
	} else if format == "trix" {
		data, err = trix.ToTrix(dn, password)
		if err != nil {
			return "", fmt.Errorf("error serializing trix: %w", err)
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

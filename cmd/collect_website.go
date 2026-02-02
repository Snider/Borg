package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
	"github.com/Snider/Borg/pkg/cache"
	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/Snider/Borg/pkg/ui"
	"github.com/Snider/Borg/pkg/website"

	"github.com/spf13/cobra"
)

// collectWebsiteCmd represents the collect website command
var collectWebsiteCmd = NewCollectWebsiteCmd()

func init() {
	GetCollectCmd().AddCommand(GetCollectWebsiteCmd())
}

func GetCollectWebsiteCmd() *cobra.Command {
	return collectWebsiteCmd
}

func NewCollectWebsiteCmd() *cobra.Command {
	collectWebsiteCmd := &cobra.Command{
		Use:   "website [url]",
		Short: "Collect a single website",
		Long:  `Collect a single website and store it in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			websiteURL := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			depth, _ := cmd.Flags().GetInt("depth")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")
			password, _ := cmd.Flags().GetString("password")

			if format != "datanode" && format != "tim" && format != "trix" {
				return fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', or 'trix')", format)
			}

			prompter := ui.NewNonInteractivePrompter(ui.GetWebsiteQuote)
			prompter.Start()
			defer prompter.Stop()
			var bar *progressbar.ProgressBar
			if prompter.IsInteractive() {
				bar = ui.NewProgressBar(-1, "Crawling website")
			}

			noCache, _ := cmd.Flags().GetBool("no-cache")

			var cacheInstance *cache.Cache
			var err error
			if !noCache {
				cacheDir, err := GetCacheDir(cmd)
				if err != nil {
					return err
				}
				cacheInstance, err = cache.New(cacheDir)
				if err != nil {
					return fmt.Errorf("failed to create cache: %w", err)
				}
			}

			dn, err := website.DownloadAndPackageWebsite(websiteURL, depth, bar, cacheInstance)
			if err != nil {
				return fmt.Errorf("error downloading and packaging website: %w", err)
			}

			if cacheInstance != nil {
				if err := cacheInstance.Close(); err != nil {
					return fmt.Errorf("failed to close cache: %w", err)
				}
			}

			var data []byte
			if format == "tim" {
				tim, err := tim.FromDataNode(dn)
				if err != nil {
					return fmt.Errorf("error creating tim: %w", err)
				}
				data, err = tim.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing tim: %w", err)
				}
			} else if format == "trix" {
				data, err = trix.ToTrix(dn, password)
				if err != nil {
					return fmt.Errorf("error serializing trix: %w", err)
				}
			} else {
				data, err = dn.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing DataNode: %w", err)
				}
			}

			compressedData, err := compress.Compress(data, compression)
			if err != nil {
				return fmt.Errorf("error compressing data: %w", err)
			}

			if outputFile == "" {
				outputFile = "website." + format
				if compression != "none" {
					outputFile += "." + compression
				}
			}

			err = os.WriteFile(outputFile, compressedData, 0644)
			if err != nil {
				return fmt.Errorf("error writing website to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Website saved to", outputFile)
			return nil
		},
	}
	collectWebsiteCmd.PersistentFlags().String("output", "", "Output file for the DataNode")
	collectWebsiteCmd.PersistentFlags().Int("depth", 2, "Recursion depth for downloading")
	collectWebsiteCmd.PersistentFlags().String("format", "datanode", "Output format (datanode, tim, or trix)")
	collectWebsiteCmd.PersistentFlags().String("compression", "none", "Compression format (none, gz, or xz)")
	collectWebsiteCmd.PersistentFlags().String("password", "", "Password for encryption")
	collectWebsiteCmd.Flags().Bool("no-cache", false, "Skip cache, re-download everything")
	collectWebsiteCmd.Flags().String("cache-dir", "", "Custom cache location")
	return collectWebsiteCmd
}

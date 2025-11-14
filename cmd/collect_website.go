package cmd

import (
	"fmt"
	"os"

	"github.com/schollz/progressbar/v3"
	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/tim"
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

			if format != "datanode" && format != "tim" {
				return fmt.Errorf("invalid format: %s (must be 'datanode' or 'tim')", format)
			}

			prompter := ui.NewNonInteractivePrompter(ui.GetWebsiteQuote)
			prompter.Start()
			defer prompter.Stop()
			var bar *progressbar.ProgressBar
			if prompter.IsInteractive() {
				bar = ui.NewProgressBar(-1, "Crawling website")
			}

			dn, err := website.DownloadAndPackageWebsite(websiteURL, depth, bar)
			if err != nil {
				return fmt.Errorf("error downloading and packaging website: %w", err)
			}

			var data []byte
			if format == "tim" {
				t, err := tim.FromDataNode(dn)
				if err != nil {
					return fmt.Errorf("error creating matrix: %w", err)
				}
				data, err = t.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing matrix: %w", err)
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
	collectWebsiteCmd.PersistentFlags().String("format", "datanode", "Output format (datanode or tim)")
	collectWebsiteCmd.PersistentFlags().String("compression", "none", "Compression format (none, gz, or xz)")
	return collectWebsiteCmd
}

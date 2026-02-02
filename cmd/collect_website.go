package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/Snider/Borg/pkg/compress"
	borghttp "github.com/Snider/Borg/pkg/http"
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
			rateLimit, _ := cmd.Flags().GetString("rate-limit")
			burst, _ := cmd.Flags().GetInt("burst")
			rateConfig, _ := cmd.Flags().GetString("rate-config")

			if format != "datanode" && format != "tim" && format != "trix" {
				return fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', or 'trix')", format)
			}

			config := &borghttp.Config{
				Defaults: borghttp.Rate{
					RequestsPerSecond: 10, // A reasonable default
					Burst:             10,
				},
				Domains: make(map[string]borghttp.Rate),
			}

			if rateConfig != "" {
				var err error
				config, err = borghttp.ParseConfig(rateConfig)
				if err != nil {
					return fmt.Errorf("error parsing rate config: %w", err)
				}
			}

			if rateLimit != "" {
				parts := strings.Split(rateLimit, "/")
				if len(parts) != 2 || (parts[1] != "s" && parts[1] != "m") {
					return fmt.Errorf("invalid rate limit format: %s (e.g., 2/s or 120/m)", rateLimit)
				}
				rate, err := strconv.ParseFloat(parts[0], 64)
				if err != nil {
					return fmt.Errorf("invalid rate: %w", err)
				}
				if parts[1] == "m" {
					rate = rate / 60
				}
				config.Defaults.RequestsPerSecond = rate
			}

			if burst > 0 {
				config.Defaults.Burst = burst
			}

			client := &http.Client{
				Transport: borghttp.NewRateLimitingRoundTripper(config, http.DefaultTransport),
			}

			prompter := ui.NewNonInteractivePrompter(ui.GetWebsiteQuote)
			prompter.Start()
			defer prompter.Stop()
			var bar *progressbar.ProgressBar
			if prompter.IsInteractive() {
				bar = ui.NewProgressBar(-1, "Crawling website")
			}

			dn, err := website.DownloadAndPackageWebsite(websiteURL, depth, bar, client)
			if err != nil {
				return fmt.Errorf("error downloading and packaging website: %w", err)
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
	collectWebsiteCmd.Flags().String("rate-limit", "", "Requests per second (e.g., 2/s) or minute (e.g., 120/m)")
	collectWebsiteCmd.Flags().Int("burst", 0, "Burst allowance")
	collectWebsiteCmd.Flags().String("rate-config", "", "Path to a rate limit configuration file")
	return collectWebsiteCmd
}

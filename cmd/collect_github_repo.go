package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Snider/Borg/pkg/compress"
	borghttp "github.com/Snider/Borg/pkg/http"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/Snider/Borg/pkg/ui"
	"github.com/Snider/Borg/pkg/vcs"

	"github.com/spf13/cobra"
)

const (
	defaultFilePermission = 0644
)

var (
	// GitCloner is the git cloner used by the command. It can be replaced for testing.
	GitCloner = vcs.NewGitCloner()
)

// NewCollectGithubRepoCmd creates a new cobra command for collecting a single git repository.
func NewCollectGithubRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo [repository-url]",
		Short: "Collect a single Git repository",
		Long:  `Collect a single Git repository and store it in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURL := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")
			password, _ := cmd.Flags().GetString("password")
			rateLimit, _ := cmd.Flags().GetString("rate-limit")
			burst, _ := cmd.Flags().GetInt("burst")
			rateConfig, _ := cmd.Flags().GetString("rate-config")

			if format != "datanode" && format != "tim" && format != "trix" && format != "stim" {
				return fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', 'trix', or 'stim')", format)
			}
			if compression != "none" && compression != "gz" && compression != "xz" {
				return fmt.Errorf("invalid compression: %s (must be 'none', 'gz', or 'xz')", compression)
			}

			config := &borghttp.Config{
				Defaults: borghttp.Rate{
					RequestsPerSecond: 1, // GitHub API has strict limits
					Burst:             1,
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
			cloner := vcs.NewGitClonerWithClient(client)

			prompter := ui.NewNonInteractivePrompter(ui.GetVCSQuote)
			prompter.Start()
			defer prompter.Stop()

			var progressWriter io.Writer
			if prompter.IsInteractive() {
				bar := ui.NewProgressBar(-1, "Cloning repository")
				progressWriter = ui.NewProgressWriter(bar)
			}

			dn, err := cloner.CloneGitRepository(repoURL, progressWriter)
			if err != nil {
				return fmt.Errorf("error cloning repository: %w", err)
			}

			var data []byte
			if format == "tim" {
				t, err := tim.FromDataNode(dn)
				if err != nil {
					return fmt.Errorf("error creating tim: %w", err)
				}
				data, err = t.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing tim: %w", err)
				}
			} else if format == "stim" {
				if password == "" {
					return fmt.Errorf("password required for stim format")
				}
				t, err := tim.FromDataNode(dn)
				if err != nil {
					return fmt.Errorf("error creating tim: %w", err)
				}
				data, err = t.ToSigil(password)
				if err != nil {
					return fmt.Errorf("error encrypting stim: %w", err)
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
				outputFile = "repo." + format
				if compression != "none" {
					outputFile += "." + compression
				}
			}

			err = os.WriteFile(outputFile, compressedData, defaultFilePermission)
			if err != nil {
				return fmt.Errorf("error writing DataNode to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Repository saved to", outputFile)
			return nil
		},
	}
	cmd.Flags().String("output", "", "Output file for the DataNode")
	cmd.Flags().String("format", "datanode", "Output format (datanode, tim, trix, or stim)")
	cmd.Flags().String("compression", "none", "Compression format (none, gz, or xz)")
	cmd.Flags().String("password", "", "Password for encryption (required for trix/stim)")
	cmd.Flags().String("rate-limit", "", "Requests per second (e.g., 2/s) or minute (e.g., 120/m)")
	cmd.Flags().Int("burst", 0, "Burst allowance")
	cmd.Flags().String("rate-config", "", "Path to a rate limit configuration file")
	return cmd
}

func init() {
	collectGithubCmd.AddCommand(NewCollectGithubRepoCmd())
}

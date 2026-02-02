package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/Snider/Borg/pkg/ui"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	// GithubClient is the github client used by the command. It can be replaced for testing.
	GithubClient = github.NewGithubClient()
)

var collectGithubReposCmd = &cobra.Command{
	Use:   "repos [user-or-org]",
	Short: "Collects all public repositories for a user or organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parallel, _ := cmd.Flags().GetInt("parallel")
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")
		compression, _ := cmd.Flags().GetString("compression")
		password, _ := cmd.Flags().GetString("password")

		repos, err := GithubClient.GetPublicRepos(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		prompter := ui.NewNonInteractivePrompter(ui.GetVCSQuote)
		prompter.Start()
		defer prompter.Stop()
		var bar *progressbar.ProgressBar
		if prompter.IsInteractive() {
			bar = ui.NewProgressBar(len(repos), "Cloning repositories")
		}

		downloader := github.NewDownloader(parallel, bar)
		dn, err := downloader.DownloadRepositories(cmd.Context(), repos)
		if err != nil {
			return err
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
			outputFile = args[0] + "." + format
			if compression != "none" {
				outputFile += "." + compression
			}
		}

		err = os.WriteFile(outputFile, compressedData, 0644)
		if err != nil {
			return fmt.Errorf("error writing repos to file: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Repositories saved to", outputFile)
		return nil
	},
}

func init() {
	collectGithubCmd.AddCommand(collectGithubReposCmd)
	collectGithubReposCmd.PersistentFlags().Int("parallel", 1, "Number of concurrent workers")
	collectGithubReposCmd.PersistentFlags().String("output", "", "Output file for the DataNode")
	collectGithubReposCmd.PersistentFlags().String("format", "datanode", "Output format (datanode, tim, or trix)")
	collectGithubReposCmd.PersistentFlags().String("compression", "none", "Compression format (none, gz, or xz)")
	collectGithubReposCmd.PersistentFlags().String("password", "", "Password for encryption")
}

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/ui"
	"github.com/spf13/cobra"
)

// NewCollectGithubIssuesCmd creates a new cobra command for collecting github issues.
func NewCollectGithubIssuesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issues <owner/repo>",
		Short: "Collect issues from a GitHub repository",
		Long:  `Collect all issues from a GitHub repository and store them in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath := args[0]
			parts := strings.Split(repoPath, "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository path: %s (must be in the format <owner>/<repo>)", repoPath)
			}
			owner, repo := parts[0], parts[1]

			outputFile, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")

			if format != "datanode" {
				return fmt.Errorf("invalid format: %s (must be 'datanode')", format)
			}
			if compression != "none" && compression != "gz" && compression != "xz" {
				return fmt.Errorf("invalid compression: %s (must be 'none', 'gz', or 'xz')", compression)
			}

			prompter := ui.NewNonInteractivePrompter(ui.GetVCSQuote)
			prompter.Start()
			defer prompter.Stop()

			client := github.NewGithubClient()
			dn, err := client.GetIssues(cmd.Context(), owner, repo)
			if err != nil {
				return fmt.Errorf("error getting issues: %w", err)
			}

			data, err := dn.ToTar()
			if err != nil {
				return fmt.Errorf("error serializing DataNode: %w", err)
			}

			compressedData, err := compress.Compress(data, compression)
			if err != nil {
				return fmt.Errorf("error compressing data: %w", err)
			}

			if outputFile == "" {
				outputFile = "issues." + format
				if compression != "none" {
					outputFile += "." + compression
				}
			}

			err = os.WriteFile(outputFile, compressedData, 0644)
			if err != nil {
				return fmt.Errorf("error writing DataNode to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Issues saved to", outputFile)
			return nil
		},
	}
	cmd.Flags().String("output", "", "Output file for the DataNode")
	cmd.Flags().String("format", "datanode", "Output format (datanode)")
	cmd.Flags().String("compression", "none", "Compression format (none, gz, or xz)")
	return cmd
}

func init() {
	GetCollectGithubCmd().AddCommand(NewCollectGithubIssuesCmd())
}

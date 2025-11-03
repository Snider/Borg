package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/matrix"
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

			if format != "datanode" && format != "matrix" {
				return fmt.Errorf("invalid format: %s (must be 'datanode' or 'matrix')", format)
			}
			if compression != "none" && compression != "gz" && compression != "xz" {
				return fmt.Errorf("invalid compression: %s (must be 'none', 'gz', or 'xz')", compression)
			}

			prompter := ui.NewNonInteractivePrompter(ui.GetVCSQuote)
			prompter.Start()
			defer prompter.Stop()

			var progressWriter io.Writer
			if prompter.IsInteractive() {
				bar := ui.NewProgressBar(-1, "Cloning repository")
				progressWriter = ui.NewProgressWriter(bar)
			}

			dn, err := GitCloner.CloneGitRepository(repoURL, progressWriter)
			if err != nil {
				return fmt.Errorf("error cloning repository: %w", err)
			}

			var data []byte
			if format == "matrix" {
				matrix, err := matrix.FromDataNode(dn)
				if err != nil {
					return fmt.Errorf("error creating matrix: %w", err)
				}
				data, err = matrix.ToTar()
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
	cmd.Flags().String("format", "datanode", "Output format (datanode or matrix)")
	cmd.Flags().String("compression", "none", "Compression format (none, gz, or xz)")
	return cmd
}

func init() {
	collectGithubCmd.AddCommand(NewCollectGithubRepoCmd())
}

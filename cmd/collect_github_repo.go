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

// collectGithubRepoCmd represents the collect github repo command
var collectGithubRepoCmd = &cobra.Command{
	Use:   "repo [repository-url]",
	Short: "Collect a single Git repository",
	Long:  `Collect a single Git repository and store it in a DataNode.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoURL := args[0]
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")
		compression, _ := cmd.Flags().GetString("compression")

		prompter := ui.NewNonInteractivePrompter(ui.GetVCSQuote)
		prompter.Start()
		defer prompter.Stop()

		var progressWriter io.Writer
		if prompter.IsInteractive() {
			bar := ui.NewProgressBar(-1, "Cloning repository")
			progressWriter = ui.NewProgressWriter(bar)
		}

		dn, err := vcs.CloneGitRepository(repoURL, progressWriter)
		if err != nil {
			fmt.Printf("Error cloning repository: %v\n", err)
			return
		}

		var data []byte
		if format == "matrix" {
			matrix, err := matrix.FromDataNode(dn)
			if err != nil {
				fmt.Printf("Error creating matrix: %v\n", err)
				return
			}
			data, err = matrix.ToTar()
			if err != nil {
				fmt.Printf("Error serializing matrix: %v\n", err)
				return
			}
		} else {
			data, err = dn.ToTar()
			if err != nil {
				fmt.Printf("Error serializing DataNode: %v\n", err)
				return
			}
		}

		compressedData, err := compress.Compress(data, compression)
		if err != nil {
			fmt.Printf("Error compressing data: %v\n", err)
			return
		}

		if outputFile == "" {
			outputFile = "repo." + format
			if compression != "none" {
				outputFile += "." + compression
			}
		}

		err = os.WriteFile(outputFile, compressedData, 0644)
		if err != nil {
			fmt.Printf("Error writing DataNode to file: %v\n", err)
			return
		}

		fmt.Printf("Repository saved to %s\n", outputFile)
	},
}

// init registers the 'repo' subcommand and its flags under the GitHub command.
func init() {
	collectGithubCmd.AddCommand(collectGithubRepoCmd)
	collectGithubRepoCmd.PersistentFlags().String("output", "", "Output file for the DataNode")
	collectGithubRepoCmd.PersistentFlags().String("format", "datanode", "Output format (datanode or matrix)")
	collectGithubRepoCmd.PersistentFlags().String("compression", "none", "Compression format (none, gz, or xz)")
}

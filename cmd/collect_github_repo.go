package cmd

import (
	"fmt"
	"os"

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

		bar := ui.NewProgressBar(-1, "Cloning repository")
		defer bar.Finish()

		dn, err := vcs.CloneGitRepository(repoURL, bar)
		if err != nil {
			fmt.Printf("Error cloning repository: %v\n", err)
			return
		}

		data, err := dn.ToTar()
		if err != nil {
			fmt.Printf("Error serializing DataNode: %v\n", err)
			return
		}

		err = os.WriteFile(outputFile, data, 0644)
		if err != nil {
			fmt.Printf("Error writing DataNode to file: %v\n", err)
			return
		}

		fmt.Printf("Repository saved to %s\n", outputFile)
	},
}

func init() {
	collectGithubCmd.AddCommand(collectGithubRepoCmd)
	collectGithubRepoCmd.PersistentFlags().String("output", "repo.dat", "Output file for the DataNode")
}

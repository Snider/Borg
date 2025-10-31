package cmd

import (
	"fmt"
	"os"

	"borg-data-collector/pkg/vcs"

	"github.com/spf13/cobra"
)

// collectCmd represents the collect command
var collectCmd = &cobra.Command{
	Use:   "collect [repository-url]",
	Short: "Collect a single repository",
	Long:  `Collect a single repository and store it in a DataNode.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoURL := args[0]
		outputFile, _ := cmd.Flags().GetString("output")

		dn, err := vcs.CloneGitRepository(repoURL)
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
	rootCmd.AddCommand(collectCmd)
	collectCmd.PersistentFlags().String("output", "repo.dat", "Output file for the DataNode")
}

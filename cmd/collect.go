package cmd

import (
	"fmt"

	"borg-data-collector/pkg/trix"

	"github.com/spf13/cobra"
)

// collectCmd represents the collect command
var collectCmd = &cobra.Command{
	Use:   "collect [repository-url]",
	Short: "Collect a single repository",
	Long: `Collect a single repository and store it in a Trix cube.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Please provide a repository URL")
			return
		}
		repoURL := args[0]
		clonePath, _ := cmd.Flags().GetString("path")
		outputFile, _ := cmd.Flags().GetString("output")

		cube, err := trix.NewCube(outputFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer cube.Close()

		err = addRepoToCube(repoURL, cube, clonePath)
		if err != nil {
			fmt.Println(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(collectCmd)
	collectCmd.PersistentFlags().String("path", "/tmp/borg-clone", "Path to clone the repository")
	collectCmd.PersistentFlags().String("output", "borg.cube", "Output file for the Trix cube")
}

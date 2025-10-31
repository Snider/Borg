package cmd

import (
	"fmt"
	"os"

	"borg-data-collector/pkg/borg"
	"borg-data-collector/pkg/github"
	"borg-data-collector/pkg/trix"

	"github.com/spf13/cobra"
)

// allCmd represents the all command
var allCmd = &cobra.Command{
	Use:   "all [user/org]",
	Short: "Collect all public repositories from a user or organization",
	Long: `Collect all public repositories from a user or organization and store them in a Trix cube.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(borg.GetRandomAssimilationMessage())

		repos, err := github.GetPublicRepos(args[0])
		if err != nil {
			fmt.Println(err)
			return
		}

		outputFile, _ := cmd.Flags().GetString("output")

		cube, err := trix.NewCube(outputFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer cube.Close()

		for _, repoURL := range repos {
			fmt.Printf("Cloning %s...\n", repoURL)

			tempPath, err := os.MkdirTemp("", "borg-clone-*")
			if err != nil {
				fmt.Println(err)
				return
			}
			defer os.RemoveAll(tempPath)

			err = addRepoToCube(repoURL, cube, tempPath)
			if err != nil {
				fmt.Printf("Error cloning %s: %s\n", repoURL, err)
				continue
			}
		}

		fmt.Println(borg.GetRandomCodeLongMessage())
	},
}

func init() {
	collectCmd.AddCommand(allCmd)
}

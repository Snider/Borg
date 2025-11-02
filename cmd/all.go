package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/vcs"

	"github.com/spf13/cobra"
)

// allCmd represents the all command
var allCmd = &cobra.Command{
	Use:   "all [user/org]",
	Short: "Collect all public repositories from a user or organization",
	Long:  `Collect all public repositories from a user or organization and store them in a DataNode.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repos, err := github.GetPublicRepos(context.Background(), args[0])
		if err != nil {
			fmt.Println(err)
			return
		}

		outputDir, _ := cmd.Flags().GetString("output")

		for _, repoURL := range repos {
			fmt.Printf("Cloning %s...\n", repoURL)

			dn, err := vcs.CloneGitRepository(repoURL)
			if err != nil {
				fmt.Printf("Error cloning %s: %s\n", repoURL, err)
				continue
			}

			data, err := dn.ToTar()
			if err != nil {
				fmt.Printf("Error serializing DataNode for %s: %v\n", repoURL, err)
				continue
			}

			repoName := strings.Split(repoURL, "/")[len(strings.Split(repoURL, "/"))-1]
			outputFile := fmt.Sprintf("%s/%s.dat", outputDir, repoName)
			err = os.WriteFile(outputFile, data, 0644)
			if err != nil {
				fmt.Printf("Error writing DataNode for %s to file: %v\n", repoURL, err)
				continue
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(allCmd)
	allCmd.PersistentFlags().String("output", ".", "Output directory for the DataNodes")
}

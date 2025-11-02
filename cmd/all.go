package cmd

import (
	"context"
	"fmt"
	"log/slog"
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
		log := cmd.Context().Value("logger").(*slog.Logger)
		repos, err := github.GetPublicRepos(context.Background(), args[0])
		if err != nil {
			log.Error("failed to get public repos", "err", err)
			return
		}

		outputDir, _ := cmd.Flags().GetString("output")

		for _, repoURL := range repos {
			log.Info("cloning repository", "url", repoURL)

			dn, err := vcs.CloneGitRepository(repoURL)
			if err != nil {
				log.Error("failed to clone repository", "url", repoURL, "err", err)
				continue
			}

			data, err := dn.ToTar()
			if err != nil {
				log.Error("failed to serialize datanode", "url", repoURL, "err", err)
				continue
			}

			repoName := strings.Split(repoURL, "/")[len(strings.Split(repoURL, "/"))-1]
			outputFile := fmt.Sprintf("%s/%s.dat", outputDir, repoName)
			err = os.WriteFile(outputFile, data, 0644)
			if err != nil {
				log.Error("failed to write datanode to file", "url", repoURL, "err", err)
				continue
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(allCmd)
	allCmd.PersistentFlags().String("output", ".", "Output directory for the DataNodes")
}

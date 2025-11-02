package cmd

import (
	"fmt"

	"github.com/Snider/Borg/pkg/github"
	"github.com/spf13/cobra"
)

var collectGithubReposCmd = &cobra.Command{
	Use:   "repos [user-or-org]",
	Short: "Collects all public repositories for a user or organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := github.GetPublicRepos(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		for _, repo := range repos {
			fmt.Println(repo)
		}
		return nil
	},
}

func init() {
	collectGithubCmd.AddCommand(collectGithubReposCmd)
}

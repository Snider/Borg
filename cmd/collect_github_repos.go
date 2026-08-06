package cmd

import (
	"fmt"

	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/httpclient"
	"github.com/spf13/cobra"
)

var collectGithubReposCmd = &cobra.Command{
	Use:   "repos [user-or-org]",
	Short: "Collects all public repositories for a user or organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		totalTimeout, _ := cmd.Flags().GetDuration("timeout")
		connectTimeout, _ := cmd.Flags().GetDuration("connect-timeout")
		tlsTimeout, _ := cmd.Flags().GetDuration("tls-timeout")
		headerTimeout, _ := cmd.Flags().GetDuration("header-timeout")

		httpClient := httpclient.NewClient(totalTimeout, connectTimeout, tlsTimeout, headerTimeout)
		githubClient := github.NewGithubClient(httpClient)

		repos, err := githubClient.GetPublicRepos(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		for _, repo := range repos {
			fmt.Fprintln(cmd.OutOrStdout(), repo)
		}
		return nil
	},
}

func init() {
	collectGithubCmd.AddCommand(collectGithubReposCmd)
}

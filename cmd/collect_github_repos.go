package cmd

import (
	"fmt"
	"net/http"

	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/ratelimit"
	"github.com/spf13/cobra"
)

var (
	// GithubClient is the github client used by the command. It can be replaced for testing.
	GithubClient = func(client *http.Client) github.GithubClient {
		return github.NewGithubClient(client)
	}
)

var collectGithubReposCmd = &cobra.Command{
	Use:   "repos [user-or-org]",
	Short: "Collects all public repositories for a user or organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bandwidth, _ := cmd.Flags().GetString("bandwidth")
		bytesPerSec, err := ratelimit.ParseBandwidth(bandwidth)
		if err != nil {
			return fmt.Errorf("invalid bandwidth: %w", err)
		}

		client := github.NewAuthenticatedClient(cmd.Context(), ratelimit.NewRateLimitedRoundTripper(http.DefaultTransport, bytesPerSec))
		githubClient := GithubClient(client)

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

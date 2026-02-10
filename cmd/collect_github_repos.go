package cmd

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Snider/Borg/pkg/github"
	borghttp "github.com/Snider/Borg/pkg/http"
	"github.com/spf13/cobra"
)

var (
	// GithubClient is the github client used by the command. It can be replaced for testing.
	GithubClient = github.NewGithubClient(nil)
)

var collectGithubReposCmd = &cobra.Command{
	Use:   "repos [user-or-org]",
	Short: "Collects all public repositories for a user or organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rateLimit, _ := cmd.Flags().GetString("rate-limit")
		burst, _ := cmd.Flags().GetInt("burst")
		rateConfig, _ := cmd.Flags().GetString("rate-config")

		config := &borghttp.Config{
			Defaults: borghttp.Rate{
				RequestsPerSecond: 1, // GitHub API has strict limits
				Burst:             1,
			},
			Domains: make(map[string]borghttp.Rate),
		}

		if rateConfig != "" {
			var err error
			config, err = borghttp.ParseConfig(rateConfig)
			if err != nil {
				return fmt.Errorf("error parsing rate config: %w", err)
			}
		}

		if rateLimit != "" {
			parts := strings.Split(rateLimit, "/")
			if len(parts) != 2 || (parts[1] != "s" && parts[1] != "m") {
				return fmt.Errorf("invalid rate limit format: %s (e.g., 2/s or 120/m)", rateLimit)
			}
			rate, err := strconv.ParseFloat(parts[0], 64)
			if err != nil {
				return fmt.Errorf("invalid rate: %w", err)
			}
			if parts[1] == "m" {
				rate = rate / 60
			}
			config.Defaults.RequestsPerSecond = rate
		}

		if burst > 0 {
			config.Defaults.Burst = burst
		}

		client := &http.Client{
			Transport: borghttp.NewRateLimitingRoundTripper(config, http.DefaultTransport),
		}
		ghClient := github.NewGithubClient(client)

		repos, err := ghClient.GetPublicRepos(cmd.Context(), args[0])
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
	collectGithubReposCmd.Flags().String("rate-limit", "", "Requests per second (e.g., 2/s) or minute (e.g., 120/m)")
	collectGithubReposCmd.Flags().Int("burst", 0, "Burst allowance")
	collectGithubReposCmd.Flags().String("rate-config", "", "Path to a rate limit configuration file")
}

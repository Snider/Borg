package cmd

import (
	"fmt"
	"strings"

	"github.com/Snider/Borg/pkg/failures"
	"github.com/Snider/Borg/pkg/github"
	"github.com/spf13/cobra"
)

var (
	// GithubClient is the github client used by the command. It can be replaced for testing.
	GithubClient = github.NewGithubClient()
)

var collectGithubReposCmd = &cobra.Command{
	Use:   "repos [user-or-org]",
	Short: "Collects all public repositories for a user or organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		failuresDir, _ := cmd.Flags().GetString("failures-dir")
		onFailure, _ := cmd.Flags().GetString("on-failure")

		manager, err := failures.NewManager(failuresDir, "github:repos:"+args[0])
		if err != nil {
			return fmt.Errorf("failed to create failure manager: %w", err)
		}
		defer manager.Finalize()

		repos, err := GithubClient.GetPublicRepos(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		manager.SetTotal(len(repos))

		attempts := make(map[string]int)
		for i := 0; i < len(repos); i++ {
			repo := repos[i]
			attempts[repo]++

			fmt.Fprintln(cmd.OutOrStdout(), "Collecting", repo)
			err := collectRepo(repo, "", "datanode", "none", "", cmd)
			if err != nil {
				retryable := !strings.Contains(err.Error(), "not found")
				manager.RecordFailure(&failures.Failure{
					URL:       repo,
					Error:     err.Error(),
					Retryable: retryable,
					Attempts:  attempts[repo],
				})

				if onFailure == "stop" {
					return fmt.Errorf("stopping on first failure: %w", err)
				} else if onFailure == "prompt" {
					fmt.Printf("Failed to collect %s. Would you like to (c)ontinue, (s)top, or (r)etry? ", repo)
					var response string
					fmt.Scanln(&response)
					switch response {
					case "s":
						return fmt.Errorf("stopping on user prompt")
					case "r":
						i-- // Retry the same repo
						continue
					default:
						// Continue
					}
				}
			}
		}

		return nil
	},
}

func init() {
	collectGithubCmd.AddCommand(collectGithubReposCmd)
}

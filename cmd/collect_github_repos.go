package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/Snider/Borg/pkg/events"
	"github.com/Snider/Borg/pkg/github"
	"github.com/spf13/cobra"
)

var (
	// GithubClient is the github client used by the command. It can be replaced for testing.
	GithubClient = github.NewGithubClient()
	eventsFlag   bool
	webhook      string
	eventLog     string
)

var collectGithubReposCmd = &cobra.Command{
	Use:   "repos [user-or-org]",
	Short: "Collects all public repositories for a user or organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var stdout io.Writer
		if eventsFlag {
			stdout = cmd.OutOrStdout()
		}
		emitter, err := events.NewEventEmitter(stdout, webhook, eventLog)
		if err != nil {
			return err
		}
		defer emitter.Close()

		repos, err := GithubClient.GetPublicRepos(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if err := emitter.Emit(events.CollectionStarted, events.CollectionStartedData{
			Source:         args[0],
			EstimatedItems: len(repos),
		}); err != nil {
			return err
		}

		startTime := time.Now()
		for i, repo := range repos {
			itemStartTime := time.Now()
			if err := emitter.Emit(events.ItemStarted, events.ItemStartedData{
				URL:   repo,
				Index: i + 1,
				Total: len(repos),
			}); err != nil {
				return err
			}

			if !eventsFlag {
				fmt.Fprintln(cmd.OutOrStdout(), repo)
			}

			if err := emitter.Emit(events.ItemCompleted, events.ItemCompletedData{
				URL:        repo,
				Size:       0, // Not applicable here
				DurationMs: time.Since(itemStartTime).Milliseconds(),
				Index:      i + 1,
				Total:      len(repos),
			}); err != nil {
				return err
			}
		}

		if err := emitter.Emit(events.CollectionCompleted, events.CollectionCompletedData{
			Stats:      map[string]interface{}{"repos": len(repos)},
			DurationMs: time.Since(startTime).Milliseconds(),
		}); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	collectGithubCmd.AddCommand(collectGithubReposCmd)
	collectGithubReposCmd.Flags().BoolVar(&eventsFlag, "events", false, "Output JSON events to stdout")
	collectGithubReposCmd.Flags().StringVar(&webhook, "webhook", "", "Webhook URL to send event notifications")
	collectGithubReposCmd.Flags().StringVar(&eventLog, "event-log", "", "Log file for events")
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/bitcointalk"
	"github.com/spf13/cobra"
)

// collectBitcoinTalkCmd represents the collect bitcointalk command
var collectBitcoinTalkCmd = NewCollectBitcoinTalkCmd()

func init() {
	GetCollectCmd().AddCommand(GetCollectBitcoinTalkCmd())
}

func NewCollectBitcoinTalkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bitcointalk",
		Short: "Collect a resource from bitcointalk.org.",
		Long:  `Collect a resource from bitcointalk.org and store it in a DataNode.`,
	}

	cmd.AddCommand(NewCollectBitcoinTalkThreadCmd())
	cmd.AddCommand(NewCollectBitcoinTalkUserCmd())
	cmd.AddCommand(NewCollectBitcoinTalkSearchCmd())

	return cmd
}

func GetCollectBitcoinTalkCmd() *cobra.Command {
	return collectBitcoinTalkCmd
}

func NewCollectBitcoinTalkThreadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "thread [thread-id]",
		Short: "Collect a single thread",
		Long:  `Collect a single thread and store it in a file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			threadID := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			if outputFile == "" {
				outputFile = fmt.Sprintf("thread-%s.md", threadID)
			}

			thread, err := bitcointalk.ScrapeThread(threadID)
			if err != nil {
				return fmt.Errorf("error scraping thread: %w", err)
			}

			var builder strings.Builder
			builder.WriteString(fmt.Sprintf("# %s\n\n", thread.Title))

			for _, post := range thread.Posts {
				builder.WriteString(fmt.Sprintf("## Author: %s\n", post.Author))
				builder.WriteString(fmt.Sprintf("**Date:** %s\n\n", post.Date))
				builder.WriteString(post.Content)
				builder.WriteString("\n\n---\n\n")
			}

			err = os.WriteFile(outputFile, []byte(builder.String()), 0644)
			if err != nil {
				return fmt.Errorf("error writing to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Thread saved to", outputFile)
			return nil
		},
	}
	cmd.PersistentFlags().String("output", "", "Output file for the thread")
	return cmd
}

func NewCollectBitcoinTalkUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user [user-id]",
		Short: "Collect a single user profile",
		Long:  `Collect a single user profile and store it in a file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			if outputFile == "" {
				outputFile = fmt.Sprintf("user-%s.json", userID)
			}

			user, err := bitcointalk.ScrapeUserPage(userID)
			if err != nil {
				return fmt.Errorf("error scraping user: %w", err)
			}

			data, err := json.MarshalIndent(user, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling user data: %w", err)
			}

			err = os.WriteFile(outputFile, data, 0644)
			if err != nil {
				return fmt.Errorf("error writing to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "User profile saved to", outputFile)
			return nil
		},
	}
	cmd.PersistentFlags().String("output", "", "Output file for the user profile")
	return cmd
}

func NewCollectBitcoinTalkSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search the forum",
		Long:  `Search the forum and store the results in a file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			if outputFile == "" {
				outputFile = "search-results.json"
			}

			results, err := bitcointalk.ScrapeSearchPage(query)
			if err != nil {
				return fmt.Errorf("error scraping search results: %w", err)
			}

			data, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling search results: %w", err)
			}

			err = os.WriteFile(outputFile, data, 0644)
			if err != nil {
				return fmt.Errorf("error writing to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Search results saved to", outputFile)
			return nil
		},
	}
	cmd.PersistentFlags().String("output", "", "Output file for the search results")
	return cmd
}

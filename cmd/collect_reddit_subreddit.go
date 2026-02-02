package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/reddit"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/spf13/cobra"
)

// collectRedditSubredditCmd represents the collect reddit subreddit command
var collectRedditSubredditCmd = NewCollectRedditSubredditCmd()

func init() {
	GetCollectRedditCmd().AddCommand(GetCollectRedditSubredditCmd())
}

func GetCollectRedditSubredditCmd() *cobra.Command {
	return collectRedditSubredditCmd
}

func NewCollectRedditSubredditCmd() *cobra.Command {
	collectRedditSubredditCmd := &cobra.Command{
		Use:   "subreddit [name]",
		Short: "Collect a subreddit's top posts",
		Long:  `Collect a subreddit's top posts and store them in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			subredditName := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			limit, _ := cmd.Flags().GetInt("limit")
			sort, _ := cmd.Flags().GetString("sort")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")
			password, _ := cmd.Flags().GetString("password")

			if format != "datanode" && format != "tim" && format != "trix" {
				return fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', or 'trix')", format)
			}

			threads, err := reddit.ScrapeSubreddit(subredditName, sort, limit)
			if err != nil {
				return fmt.Errorf("failed to scrape subreddit: %w", err)
			}

			dn := datanode.New()
			for _, threadStub := range threads {
				thread, err := reddit.ScrapeThread(threadStub.URL)
				if err != nil {
					// It's better to log the error and continue
					fmt.Fprintf(cmd.ErrOrStderr(), "failed to scrape thread %s: %v\n", threadStub.URL, err)
					continue
				}

				var builder strings.Builder
				builder.WriteString(fmt.Sprintf("# %s\n\n", thread.Title))
				builder.WriteString(fmt.Sprintf("%s\n\n", thread.Post))
				for _, comment := range thread.Comments {
					builder.WriteString(fmt.Sprintf("## %s\n\n", comment.Author))
					builder.WriteString(fmt.Sprintf("%s\n\n", comment.Body))
				}
				// Sanitize filename
				filename := strings.ReplaceAll(thread.Title, " ", "_")
				filename = strings.ReplaceAll(filename, "/", "_")
				err = dn.AddData(fmt.Sprintf("r-%s/posts/%s.md", subredditName, filename), []byte(builder.String()))
				if err != nil {
					return fmt.Errorf("error adding data to DataNode: %w", err)
				}
			}

			var data []byte
			if format == "tim" {
				tim, err := tim.FromDataNode(dn)
				if err != nil {
					return fmt.Errorf("error creating tim: %w", err)
				}
				data, err = tim.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing tim: %w", err)
				}
			} else if format == "trix" {
				data, err = trix.ToTrix(dn, password)
				if err != nil {
					return fmt.Errorf("error serializing trix: %w", err)
				}
			} else {
				data, err = dn.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing DataNode: %w", err)
				}
			}

			compressedData, err := compress.Compress(data, compression)
			if err != nil {
				return fmt.Errorf("error compressing data: %w", err)
			}

			if outputFile == "" {
				outputFile = "subreddit." + format
				if compression != "none" {
					outputFile += "." + compression
				}
			}

			err = os.WriteFile(outputFile, compressedData, 0644)
			if err != nil {
				return fmt.Errorf("error writing subreddit to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Subreddit saved to", outputFile)
			return nil
		},
	}
	collectRedditSubredditCmd.PersistentFlags().String("output", "", "Output file for the DataNode")
	collectRedditSubredditCmd.PersistentFlags().Int("limit", 100, "Number of posts to collect")
	collectRedditSubredditCmd.PersistentFlags().String("sort", "top", "Sort order for posts (top, new)")
	collectRedditSubredditCmd.PersistentFlags().String("format", "datanode", "Output format (datanode, tim, or trix)")
	collectRedditSubredditCmd.PersistentFlags().String("compression", "none", "Compression format (none, gz, or xz)")
	collectRedditSubredditCmd.PersistentFlags().String("password", "", "Password for encryption")
	return collectRedditSubredditCmd
}

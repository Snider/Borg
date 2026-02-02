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

// collectRedditUserCmd represents the collect reddit user command
var collectRedditUserCmd = NewCollectRedditUserCmd()

func init() {
	GetCollectRedditCmd().AddCommand(GetCollectRedditUserCmd())
}

func GetCollectRedditUserCmd() *cobra.Command {
	return collectRedditUserCmd
}

func NewCollectRedditUserCmd() *cobra.Command {
	collectRedditUserCmd := &cobra.Command{
		Use:   "user [name]",
		Short: "Collect a user's posts",
		Long:  `Collect a user's posts and store them in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userName := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")
			password, _ := cmd.Flags().GetString("password")

			if format != "datanode" && format != "tim" && format != "trix" {
				return fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', or 'trix')", format)
			}

			threads, err := reddit.ScrapeUser(userName)
			if err != nil {
				return fmt.Errorf("failed to scrape user: %w", err)
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
				err = dn.AddData(fmt.Sprintf("u-%s/posts/%s.md", userName, filename), []byte(builder.String()))
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
				outputFile = "user." + format
				if compression != "none" {
					outputFile += "." + compression
				}
			}

			err = os.WriteFile(outputFile, compressedData, 0644)
			if err != nil {
				return fmt.Errorf("error writing user to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "User posts saved to", outputFile)
			return nil
		},
	}
	collectRedditUserCmd.PersistentFlags().String("output", "", "Output file for the DataNode")
	collectRedditUserCmd.PersistentFlags().String("format", "datanode", "Output format (datanode, tim, or trix)")
	collectRedditUserCmd.PersistentFlags().String("compression", "none", "Compression format (none, gz, or xz)")
	collectRedditUserCmd.PersistentFlags().String("password", "", "Password for encryption")
	return collectRedditUserCmd
}

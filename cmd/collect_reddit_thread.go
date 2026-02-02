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

// collectRedditThreadCmd represents the collect reddit thread command
var collectRedditThreadCmd = NewCollectRedditThreadCmd()

func init() {
	GetCollectRedditCmd().AddCommand(GetCollectRedditThreadCmd())
}

func GetCollectRedditThreadCmd() *cobra.Command {
	return collectRedditThreadCmd
}

func NewCollectRedditThreadCmd() *cobra.Command {
	collectRedditThreadCmd := &cobra.Command{
		Use:   "thread [url]",
		Short: "Collect a single Reddit thread",
		Long:  `Collect a single Reddit thread and store it in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			threadURL := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")
			password, _ := cmd.Flags().GetString("password")

			if format != "datanode" && format != "tim" && format != "trix" {
				return fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', or 'trix')", format)
			}

			thread, err := reddit.ScrapeThread(threadURL)
			if err != nil {
				return fmt.Errorf("failed to scrape thread: %w", err)
			}

			// Convert thread to Markdown
			var builder strings.Builder
			builder.WriteString(fmt.Sprintf("# %s\n\n", thread.Title))
			builder.WriteString(fmt.Sprintf("%s\n\n", thread.Post))
			for _, comment := range thread.Comments {
				builder.WriteString(fmt.Sprintf("## %s\n\n", comment.Author))
				builder.WriteString(fmt.Sprintf("%s\n\n", comment.Body))
			}

			dn := datanode.New()
			err = dn.AddData("thread.md", []byte(builder.String()))
			if err != nil {
				return fmt.Errorf("error adding data to DataNode: %w", err)
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
				outputFile = "thread." + format
				if compression != "none" {
					outputFile += "." + compression
				}
			}

			err = os.WriteFile(outputFile, compressedData, 0644)
			if err != nil {
				return fmt.Errorf("error writing thread to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Thread saved to", outputFile)
			return nil
		},
	}
	collectRedditThreadCmd.PersistentFlags().String("output", "", "Output file for the DataNode")
	collectRedditThreadCmd.PersistentFlags().String("format", "datanode", "Output format (datanode, tim, or trix)")
	collectRedditThreadCmd.PersistentFlags().String("compression", "none", "Compression format (none, gz, or xz)")
	collectRedditThreadCmd.PersistentFlags().String("password", "", "Password for encryption")
	return collectRedditThreadCmd
}

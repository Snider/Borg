
package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/Snider/Borg/pkg/ui"
	"github.com/Snider/Borg/pkg/vcs"
	"github.com/spf13/cobra"
)

var githubAllCmd = NewGithubAllCmd()

func NewGithubAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all <owner/repo>",
		Short: "Collect all resources from a GitHub repository",
		Long:  `Collect all resources from a GitHub repository, including code, issues, and pull requests.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath := args[0]
			parts := strings.Split(repoPath, "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository path: %s (must be in the format <owner>/<repo>)", repoPath)
			}
			owner, repo := parts[0], parts[1]

			outputFile, err := cmd.Flags().GetString("output")
			if err != nil {
				return fmt.Errorf("error getting output flag: %w", err)
			}
			format, err := cmd.Flags().GetString("format")
			if err != nil {
				return fmt.Errorf("error getting format flag: %w", err)
			}
			compression, err := cmd.Flags().GetString("compression")
			if err != nil {
				return fmt.Errorf("error getting compression flag: %w", err)
			}
			password, err := cmd.Flags().GetString("password")
			if err != nil {
				return fmt.Errorf("error getting password flag: %w", err)
			}
			collectIssues, err := cmd.Flags().GetBool("issues")
			if err != nil {
				return fmt.Errorf("error getting issues flag: %w", err)
			}
			collectPRs, err := cmd.Flags().GetBool("prs")
			if err != nil {
				return fmt.Errorf("error getting prs flag: %w", err)
			}
			collectCode, err := cmd.Flags().GetBool("code")
			if err != nil {
				return fmt.Errorf("error getting code flag: %w", err)
			}

			if format != "datanode" && format != "tim" && format != "trix" {
				return fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', or 'trix')", format)
			}

			allDataNodes := datanode.New()
			prompter := ui.NewNonInteractivePrompter(ui.GetVCSQuote)
			prompter.Start()
			defer prompter.Stop()

			if collectCode {
				var progressWriter io.Writer
				if prompter.IsInteractive() {
					bar := ui.NewProgressBar(-1, "Cloning repository")
					progressWriter = ui.NewProgressWriter(bar)
				}
				cloner := vcs.NewGitCloner()
				repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
				dn, err := cloner.CloneGitRepository(repoURL, progressWriter)
				if err != nil {
					return fmt.Errorf("error cloning repository: %w", err)
				}
				if mergeErr := mergeDataNodes(allDataNodes, dn, "code"); mergeErr != nil {
					return fmt.Errorf("error merging code datanode: %w", mergeErr)
				}
			}

			client := github.NewGithubClient()
			if collectIssues {
				dn, err := client.GetIssues(cmd.Context(), owner, repo)
				if err != nil {
					return fmt.Errorf("error getting issues: %w", err)
				}
				if mergeErr := mergeDataNodes(allDataNodes, dn, ""); mergeErr != nil {
					return fmt.Errorf("error merging issues datanode: %w", mergeErr)
				}
			}

			if collectPRs {
				dn, err := client.GetPullRequests(cmd.Context(), owner, repo)
				if err != nil {
					return fmt.Errorf("error getting pull requests: %w", err)
				}
				if mergeErr := mergeDataNodes(allDataNodes, dn, ""); mergeErr != nil {
					return fmt.Errorf("error merging pull requests datanode: %w", mergeErr)
				}
			}

			var data []byte
			if format == "tim" {
				t, err := tim.FromDataNode(allDataNodes)
				if err != nil {
					return fmt.Errorf("error creating tim: %w", err)
				}
				data, err = t.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing tim: %w", err)
				}
			} else if format == "trix" {
				data, err = trix.ToTrix(allDataNodes, password)
				if err != nil {
					return fmt.Errorf("error serializing trix: %w", err)
				}
			} else {
				data, err = allDataNodes.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing DataNode: %w", err)
				}
			}

			compressedData, err := compress.Compress(data, compression)
			if err != nil {
				return fmt.Errorf("error compressing data: %w", err)
			}

			if outputFile == "" {
				outputFile = fmt.Sprintf("%s-all.%s", repo, format)
				if compression != "none" {
					outputFile += "." + compression
				}
			}

			err = os.WriteFile(outputFile, compressedData, 0644)
			if err != nil {
				return fmt.Errorf("error writing DataNode to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "All resources saved to", outputFile)
			return nil
		},
	}
	cmd.Flags().String("output", "", "Output file for the DataNode")
	cmd.Flags().String("format", "datanode", "Output format (datanode, tim, or trix)")
	cmd.Flags().String("compression", "none", "Compression format (none, gz, or xz)")
	cmd.Flags().String("password", "", "Password for encryption")
	cmd.Flags().Bool("issues", true, "Collect issues")
	cmd.Flags().Bool("prs", true, "Collect pull requests")
	cmd.Flags().Bool("code", true, "Collect code")
	return cmd
}

func GetGithubAllCmd() *cobra.Command {
	return githubAllCmd
}

func init() {
	collectGithubCmd.AddCommand(GetGithubAllCmd())
}

func mergeDataNodes(dest *datanode.DataNode, src *datanode.DataNode, prefix string) error {
	return src.Walk(".", func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !de.IsDir() {
			err := func() error {
				file, err := src.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				data, err := io.ReadAll(file)
				if err != nil {
					return err
				}
				destPath := path
				if prefix != "" {
					destPath = prefix + "/" + path
				}
				dest.AddData(destPath, data)
				return nil
			}()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

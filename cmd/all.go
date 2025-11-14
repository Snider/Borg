package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/matrix"
	"github.com/Snider/Borg/pkg/ui"
	"github.com/Snider/Borg/pkg/vcs"
	"github.com/spf13/cobra"
)

var allCmd = NewAllCmd()

func NewAllCmd() *cobra.Command {
	allCmd := &cobra.Command{
		Use:   "all [url]",
		Short: "Collect all resources from a URL",
		Long:  `Collect all resources from a URL, dispatching to the appropriate collector based on the URL type.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")

			owner, err := parseGithubOwner(url)
			if err != nil {
				return err
			}

			repos, err := GithubClient.GetPublicRepos(cmd.Context(), owner)
			if err != nil {
				return err
			}

			prompter := ui.NewNonInteractivePrompter(ui.GetVCSQuote)
			prompter.Start()
			defer prompter.Stop()

			var progressWriter io.Writer
			if prompter.IsInteractive() {
				bar := ui.NewProgressBar(len(repos), "Cloning repositories")
				progressWriter = ui.NewProgressWriter(bar)
			}

			cloner := vcs.NewGitCloner()
			allDataNodes := datanode.New()

			for _, repoURL := range repos {
				dn, err := cloner.CloneGitRepository(repoURL, progressWriter)
				if err != nil {
					// Log the error and continue
					fmt.Fprintln(cmd.ErrOrStderr(), "Error cloning repository:", err)
					continue
				}
				// This is not an efficient way to merge datanodes, but it's the only way for now
				// A better approach would be to add a Merge method to the DataNode
				repoName := strings.TrimSuffix(repoURL, ".git")
				parts := strings.Split(repoName, "/")
				repoName = parts[len(parts)-1]

				err = dn.Walk(".", func(path string, de fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !de.IsDir() {
						err := func() error {
							file, err := dn.Open(path)
							if err != nil {
								return err
							}
							defer file.Close()
							data, err := io.ReadAll(file)
							if err != nil {
								return err
							}
							allDataNodes.AddData(repoName+"/"+path, data)
							return nil
						}()
						if err != nil {
							return err
						}
					}
					return nil
				})
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "Error walking datanode:", err)
					continue
				}
			}

			var data []byte
			if format == "matrix" {
				matrix, err := matrix.FromDataNode(allDataNodes)
				if err != nil {
					return fmt.Errorf("error creating matrix: %w", err)
				}
				data, err = matrix.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing matrix: %w", err)
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

			err = os.WriteFile(outputFile, compressedData, 0644)
			if err != nil {
				return fmt.Errorf("error writing DataNode to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "All repositories saved to", outputFile)

			return nil
		},
	}
	allCmd.PersistentFlags().String("output", "all.dat", "Output file for the DataNode")
	allCmd.PersistentFlags().String("format", "datanode", "Output format (datanode or matrix)")
	allCmd.PersistentFlags().String("compression", "none", "Compression format (none, gz, or xz)")
	return allCmd
}

func GetAllCmd() *cobra.Command {
	return allCmd
}

func init() {
	RootCmd.AddCommand(GetAllCmd())
}

func parseGithubOwner(u string) (string, error) {
	owner, _, err := github.ParseRepoFromURL(u)
	if err == nil {
		return owner, nil
	}

	parsedURL, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	path := strings.Trim(parsedURL.Path, "/")
	if path == "" {
		return "", fmt.Errorf("invalid owner URL: %s", u)
	}
	parts := strings.Split(path, "/")
	if len(parts) != 1 || parts[0] == "" {
		return "", fmt.Errorf("invalid owner URL: %s", u)
	}
	return parts[0], nil
}

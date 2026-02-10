package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/progress"
	"github.com/Snider/Borg/pkg/vcs"
	"github.com/spf13/cobra"
)

var (
	// GithubClient is the github client used by the command. It can be replaced for testing.
	GithubClient = github.NewGithubClient()
	// NewGitCloner is the git cloner factory used by the command. It can be replaced for testing.
	NewGitCloner = vcs.NewGitCloner
)

var collectGithubReposCmd = &cobra.Command{
	Use:   "repos [user-or-org]",
	Short: "Collects all public repositories for a user or organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resume, _ := cmd.Flags().GetBool("resume")
		output, _ := cmd.Flags().GetString("output")
		owner := args[0]

		progressFile := ".borg-progress"
		tmpDir := fmt.Sprintf(".borg-collection-%s", owner)

		var p *progress.Progress
		var err error

		if resume {
			p, err = progress.Load(progressFile)
			if err != nil {
				return fmt.Errorf("failed to load progress file for resume: %w", err)
			}
			if p.Source != fmt.Sprintf("github:repos:%s", owner) {
				return fmt.Errorf("progress file is for a different source: %s", p.Source)
			}
			// Move failed items back to pending for retry on resume.
			p.Pending = append(p.Pending, p.Failed...)
			p.Failed = []string{}
			sort.Strings(p.Pending)
		} else {
			if _, err := os.Stat(progressFile); err == nil {
				return fmt.Errorf("found existing .borg-progress file; use --resume to continue or remove the file to start over")
			}
			if _, err := os.Stat(tmpDir); err == nil {
				return fmt.Errorf("found existing partial collection directory %s; remove it to start over", tmpDir)
			}

			repos, err := GithubClient.GetPublicRepos(cmd.Context(), owner)
			if err != nil {
				return err
			}
			sort.Strings(repos)

			p = &progress.Progress{
				Source:    fmt.Sprintf("github:repos:%s", owner),
				StartedAt: time.Now(),
				Pending:   repos,
			}
		}

		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return fmt.Errorf("failed to create tmp dir for partial results: %w", err)
		}

		if len(p.Pending) > 0 {
			if err := p.Save(progressFile); err != nil {
				return fmt.Errorf("failed to save initial progress file: %w", err)
			}
		}

		cloner := vcs.NewGitCloner()

		pendingTasks := make([]string, len(p.Pending))
		copy(pendingTasks, p.Pending)

		for _, repoFullName := range pendingTasks {
			fmt.Fprintf(cmd.OutOrStdout(), "Collecting %s...\n", repoFullName)

			repoURL := fmt.Sprintf("https://github.com/%s.git", repoFullName)
			dn, err := cloner.CloneGitRepository(repoURL, cmd.OutOrStdout())

			p.Pending = removeStringFromSlice(p.Pending, repoFullName)

			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Failed to collect %s: %v\n", repoFullName, err)
				p.Failed = append(p.Failed, repoFullName)
			} else {
				tarball, err := dn.ToTar()
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Failed to serialize datanode for %s: %v\n", repoFullName, err)
					p.Failed = append(p.Failed, repoFullName)
				} else {
					safeRepoName := strings.ReplaceAll(repoFullName, "/", "_")
					partialFile := filepath.Join(tmpDir, safeRepoName+".dat")
					if err := os.WriteFile(partialFile, tarball, 0644); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Failed to save partial result for %s: %v\n", repoFullName, err)
						p.Failed = append(p.Failed, repoFullName)
					} else {
						p.Completed = append(p.Completed, repoFullName)
					}
				}
			}

			if err := p.Save(progressFile); err != nil {
				return fmt.Errorf("CRITICAL: failed to save progress file, stopping collection: %w", err)
			}
		}

		if len(p.Pending) == 0 && len(p.Failed) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Collection complete. Merging results...")

			finalDataNode := datanode.New()
			for _, repoFullName := range p.Completed {
				safeRepoName := strings.ReplaceAll(repoFullName, "/", "_")
				partialFile := filepath.Join(tmpDir, safeRepoName+".dat")

				tarball, err := os.ReadFile(partialFile)
				if err != nil {
					return fmt.Errorf("failed to read partial result for %s: %w", repoFullName, err)
				}

				partialDN, err := datanode.FromTar(tarball)
				if err != nil {
					return fmt.Errorf("failed to parse partial result for %s: %w", repoFullName, err)
				}
				finalDataNode.Merge(partialDN)
			}

			if output == "" {
				output = fmt.Sprintf("%s-repos.dat", owner)
				fmt.Fprintf(cmd.OutOrStdout(), "No output file specified, defaulting to %s\n", output)
			}

			finalTarball, err := finalDataNode.ToTar()
			if err != nil {
				return fmt.Errorf("failed to serialize final datanode: %w", err)
			}

			if err := os.WriteFile(output, finalTarball, 0644); err != nil {
				return fmt.Errorf("failed to write final output to %s: %w", output, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully wrote collection to %s\n", output)

			fmt.Fprintln(cmd.OutOrStdout(), "Cleaning up...")
			os.Remove(progressFile)
			os.RemoveAll(tmpDir)
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Collection interrupted. Run with --resume to continue. Failed items: %d\n", len(p.Failed))
		}

		return nil
	},
}

func init() {
	collectGithubCmd.AddCommand(collectGithubReposCmd)
	collectGithubReposCmd.Flags().Bool("resume", false, "Resume collection from a .borg-progress file")
	collectGithubReposCmd.Flags().StringP("output", "o", "", "Output file name")
}

func removeStringFromSlice(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

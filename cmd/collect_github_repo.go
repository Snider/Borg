package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Snider/Borg/pkg/vcs"

	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/vcs"
	"github.com/spf13/cobra"
)

// collectGithubRepoCmd represents the collect github repo command
var collectGithubRepoCmd = &cobra.Command{
	Use:   "repo [repository-url]",
	Short: "Collect a single Git repository",
	Long:  `Collect a single Git repository and store it in a DataNode.`,
	Args:  cobra.ExactArgs(1),
// collectGitCmd represents the collect git command
var collectGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Collect one or more Git repositories",
	Long:  `Collect a single Git repository from a URL, or all public repositories from a GitHub user/organization.`,
	Run: func(cmd *cobra.Command, args []string) {
		repoURL, _ := cmd.Flags().GetString("uri")
		user, _ := cmd.Flags().GetString("user")
		output, _ := cmd.Flags().GetString("output")

		if (repoURL == "" && user == "") || (repoURL != "" && user != "") {
			fmt.Println("Error: You must specify either --uri or --user, but not both.")
			os.Exit(1)
		}

		if user != "" {
			// User specified, collect all their repos
			fmt.Printf("Fetching public repositories for %s...\n", user)
			repos, err := github.GetPublicRepos(user)
			if err != nil {
				fmt.Printf("Error fetching repositories: %v\n", err)
				return
			}
			fmt.Printf("Found %d repositories. Cloning...\n\n", len(repos))

			// Ensure output directory exists
			err = os.MkdirAll(output, 0755)
			if err != nil {
				fmt.Printf("Error creating output directory: %v\n", err)
				return
			}

			for _, repo := range repos {
				fmt.Printf("Cloning %s...\n", repo)
				dn, err := vcs.CloneGitRepository(repo)
				if err != nil {
					fmt.Printf("  Error cloning: %v\n", err)
					continue
				}

				data, err := dn.ToTar()
				if err != nil {
					fmt.Printf("  Error serializing: %v\n", err)
					continue
				}

				repoName := strings.TrimSuffix(filepath.Base(repo), ".git")
				outputFile := filepath.Join(output, fmt.Sprintf("%s.dat", repoName))
				err = os.WriteFile(outputFile, data, 0644)
				if err != nil {
					fmt.Printf("  Error writing file: %v\n", err)
					continue
				}
				fmt.Printf("  Successfully saved to %s\n", outputFile)
			}
			fmt.Println("\nCollection complete.")

		} else {
			// Single repository URL specified
			dn, err := vcs.CloneGitRepository(repoURL)
			if err != nil {
				fmt.Printf("Error cloning repository: %v\n", err)
				return
			}

			data, err := dn.ToTar()
			if err != nil {
				fmt.Printf("Error serializing DataNode: %v\n", err)
				return
			}

			err = os.WriteFile(output, data, 0644)
			if err != nil {
				fmt.Printf("Error writing DataNode to file: %v\n", err)
				return
			}
			fmt.Printf("Repository saved to %s\n", output)
		}
	},
}

func init() {
	collectGithubCmd.AddCommand(collectGithubRepoCmd)
	collectGithubRepoCmd.Flags().String("uri", "", "URL of the Git repository to collect")
	collectGitCmd.Flags().String("user", "", "GitHub user or organization to collect all repositories from")
	collectGitCmd.Flags().String("output", "repo.dat", "Output file (for --uri) or directory (for --user)")
}

package cmd

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/mocks"
	"github.com/Snider/Borg/pkg/vcs"
	"github.com/spf13/cobra"
)

func TestCollectGithubRepoCmd_Good(t *testing.T) {
	// Setup mock Git cloner
	mockCloner := &mocks.MockGitCloner{
		DN:  datanode.New(),
		Err: nil,
	}
	oldCloner := GitCloner
	GitCloner = mockCloner
	defer func() {
		GitCloner = oldCloner
	}()

	rootCmd := NewRootCmd()
	collectCmd := NewCollectCmd()
	githubCmd := GetCollectGithubCmd()
	repoCmd := NewCollectGithubRepoCmd()
	githubCmd.AddCommand(repoCmd)
	collectCmd.AddCommand(githubCmd)
	rootCmd.AddCommand(collectCmd)

	// Execute command
	out := filepath.Join(t.TempDir(), "out")
	_, err := executeCommand(rootCmd, "collect", "github", "repo", "https://github.com/testuser/repo1", "--output", out)
	if err != nil {
		t.Fatalf("collect github repo command failed: %v", err)
	}
}

func TestCollectGithubRepoCmd_Bad(t *testing.T) {
	// Setup mock Git cloner to return an error
	mockCloner := &mocks.MockGitCloner{
		DN:  nil,
		Err: fmt.Errorf("git clone error"),
	}
	oldCloner := GitCloner
	GitCloner = mockCloner
	defer func() {
		GitCloner = oldCloner
	}()

	rootCmd := NewRootCmd()
	collectCmd := NewCollectCmd()
	githubCmd := GetCollectGithubCmd()
	repoCmd := NewCollectGithubRepoCmd()
	githubCmd.AddCommand(repoCmd)
	collectCmd.AddCommand(githubCmd)
	rootCmd.AddCommand(collectCmd)

	// Execute command
	out := filepath.Join(t.TempDir(), "out")
	_, err := executeCommand(rootCmd, "collect", "github", "repo", "https://github.com/testuser/repo1", "--output", out)
	if err == nil {
		t.Fatal("expected an error, but got none")
	}
}

func TestCollectGithubRepoCmd_Ugly(t *testing.T) {
	t.Run("Invalid repo URL", func(t *testing.T) {
		rootCmd := NewRootCmd()
		collectCmd := NewCollectCmd()
		githubCmd := GetCollectGithubCmd()
		repoCmd := NewCollectGithubRepoCmd()
		githubCmd.AddCommand(repoCmd)
		collectCmd.AddCommand(githubCmd)
		rootCmd.AddCommand(collectCmd)

		repoCmd.RunE = func(cmd *cobra.Command, args []string) error {
			cloner := vcs.NewGitClonerWithClient(nil)
			_, err := cloner.CloneGitRepository(args[0], nil)
			return err
		}

		_, err := executeCommand(rootCmd, "collect", "github", "repo", "not-a-github-url")
		if err == nil {
			t.Fatal("expected an error for invalid repo URL, but got none")
		}
	})
}

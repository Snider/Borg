package cmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/mocks"
)

func TestAllCmd_Good(t *testing.T) {
	// Setup mock HTTP client for GitHub API
	mockGithubClient := &mocks.MockGithubClient{
		Repos: []string{"https://github.com/testuser/repo1.git"},
		Err:   nil,
	}
	oldGithubClient := GithubClient
	GithubClient = func(client *http.Client) github.GithubClient {
		return mockGithubClient
	}
	defer func() {
		GithubClient = oldGithubClient
	}()

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
	rootCmd.AddCommand(GetAllCmd())

	// Execute command
	out := filepath.Join(t.TempDir(), "out")
	_, err := executeCommand(rootCmd, "all", "https://github.com/testuser", "--output", out)
	if err != nil {
		t.Fatalf("all command failed: %v", err)
	}
}

func TestAllCmd_Bad(t *testing.T) {
	// Setup mock HTTP client to return an error
	mockGithubClient := &mocks.MockGithubClient{
		Repos: nil,
		Err:   fmt.Errorf("github error"),
	}
	oldGithubClient := GithubClient
	GithubClient = func(client *http.Client) github.GithubClient {
		return mockGithubClient
	}
	defer func() {
		GithubClient = oldGithubClient
	}()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetAllCmd())

	// Execute command
	out := filepath.Join(t.TempDir(), "out")
	_, err := executeCommand(rootCmd, "all", "https://github.com/baduser", "--output", out)
	if err == nil {
		t.Fatal("expected an error, but got none")
	}
}

func TestAllCmd_Ugly(t *testing.T) {
	t.Run("User with no repos", func(t *testing.T) {
		// Setup mock HTTP client for a user with no repos
		mockGithubClient := &mocks.MockGithubClient{
			Repos: []string{},
			Err:   nil,
		}
		oldGithubClient := GithubClient
		GithubClient = func(client *http.Client) github.GithubClient {
			return mockGithubClient
		}
		defer func() {
			GithubClient = oldGithubClient
		}()

		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetAllCmd())

		// Execute command
		out := filepath.Join(t.TempDir(), "out")
		_, err := executeCommand(rootCmd, "all", "https://github.com/emptyuser", "--output", out)
		if err != nil {
			t.Fatalf("all command failed for user with no repos: %v", err)
		}
	})
}

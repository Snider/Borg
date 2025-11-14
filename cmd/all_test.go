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
	mockGithubClient := mocks.NewMockClient(map[string]*http.Response{
		"https://api.github.com/users/testuser/repos": {
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(`[{"clone_url": "https://github.com/testuser/repo1.git"}]`)),
		},
	})
	oldNewAuthenticatedClient := github.NewAuthenticatedClient
	github.NewAuthenticatedClient = func(ctx context.Context) *http.Client {
		return mockGithubClient
	}
	defer func() {
		github.NewAuthenticatedClient = oldNewAuthenticatedClient
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
	mockGithubClient := mocks.NewMockClient(map[string]*http.Response{
		"https://api.github.com/users/baduser/repos": {
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Not Found"}`)),
		},
		"https://api.github.com/orgs/baduser/repos": {
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Not Found"}`)),
		},
	})
	oldNewAuthenticatedClient := github.NewAuthenticatedClient
	github.NewAuthenticatedClient = func(ctx context.Context) *http.Client {
		return mockGithubClient
	}
	defer func() {
		github.NewAuthenticatedClient = oldNewAuthenticatedClient
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
		mockGithubClient := mocks.NewMockClient(map[string]*http.Response{
			"https://api.github.com/users/emptyuser/repos": {
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`[]`)),
			},
		})
		oldNewAuthenticatedClient := github.NewAuthenticatedClient
		github.NewAuthenticatedClient = func(ctx context.Context) *http.Client {
			return mockGithubClient
		}
		defer func() {
			github.NewAuthenticatedClient = oldNewAuthenticatedClient
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

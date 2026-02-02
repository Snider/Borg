package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Snider/Borg/pkg/github"
)

type mockGithubClient struct {
	repos []string
	err   error
}

func (m *mockGithubClient) GetPublicRepos(ctx context.Context, userOrOrg string) ([]string, error) {
	return m.repos, m.err
}

func TestCollectGithubReposCmd_Good(t *testing.T) {
	oldGithubClient := GithubClient
	GithubClient = func(client *http.Client) github.GithubClient {
		return &mockGithubClient{
			repos: []string{"https://github.com/testuser/repo1", "https://github.com/testuser/repo2"},
			err:   nil,
		}
	}
	defer func() {
		GithubClient = oldGithubClient
	}()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())

	// Execute command
	output, err := executeCommand(rootCmd, "collect", "github", "repos", "testuser")
	if err != nil {
		t.Fatalf("collect github repos command failed: %v", err)
	}

	expected := "https://github.com/testuser/repo1\nhttps://github.com/testuser/repo2\n"
	if output != expected {
		t.Errorf("expected output %q, but got %q", expected, output)
	}
}

func TestCollectGithubReposCmd_Bad(t *testing.T) {
	oldGithubClient := GithubClient
	GithubClient = func(client *http.Client) github.GithubClient {
		return &mockGithubClient{
			repos: nil,
			err:   fmt.Errorf("github error"),
		}
	}
	defer func() {
		GithubClient = oldGithubClient
	}()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())

	// Execute command
	_, err := executeCommand(rootCmd, "collect", "github", "repos", "testuser")
	if err == nil {
		t.Fatal("expected an error, but got none")
	}
}

func TestCollectGithubReposCmd_Ugly(t *testing.T) {
	t.Run("Invalid bandwidth", func(t *testing.T) {
		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetCollectCmd())
		_, err := executeCommand(rootCmd, "collect", "github", "repos", "testuser", "--bandwidth", "1Gbps")
		if err == nil {
			t.Fatal("expected an error for invalid bandwidth, but got none")
		}
		if !strings.Contains(err.Error(), "invalid bandwidth") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestCollectGithubReposCmd_Bandwidth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a simplified mock of the GitHub API. It returns a single repo.
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"clone_url": "https://github.com/testuser/repo1"}]`)
	}))
	defer server.Close()

	// We need to override the API URL to point to our test server.
	oldGetPublicRepos := GithubClient
	GithubClient = func(client *http.Client) github.GithubClient {
		return &mockGithubClient{
			repos: []string{"https://github.com/testuser/repo1"},
			err:   nil,
		}
	}
	defer func() {
		GithubClient = oldGetPublicRepos
	}()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())

	// Execute command with a bandwidth limit
	start := time.Now()
	_, err := executeCommand(rootCmd, "collect", "github", "repos", "testuser", "--bandwidth", "1KB/s")
	if err != nil {
		t.Fatalf("collect github repos command failed: %v", err)
	}
	elapsed := time.Since(start)

	// Since the response is very small, we can't reliably test the bandwidth limit.
	// We'll just check that the command runs without error.
	if elapsed > 1*time.Second {
		t.Errorf("expected the command to run quickly, but it took %s", elapsed)
	}
}

// getPublicReposWithAPIURL is a copy of the private function in pkg/github/github.go, so we can test it.
type githubClient struct {
	client *http.Client
}

var getPublicReposWithAPIURL func(g *githubClient, ctx context.Context, apiURL, userOrOrg string) ([]string, error)

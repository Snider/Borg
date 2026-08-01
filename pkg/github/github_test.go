package github

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/mocks"
)

func TestGetPublicRepos_Good(t *testing.T) {
	t.Run("User Repos", func(t *testing.T) {
		mockClient := mocks.NewMockClient(map[string]*http.Response{
			"https://api.github.com/users/testuser/repos": {
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`[{"clone_url": "https://github.com/testuser/repo1.git"}]`)),
			},
		})
		client := setupMockClient(t, mockClient)
		repos, err := client.getPublicReposWithAPIURL(context.Background(), "https://api.github.com", "testuser")
		if err != nil {
			t.Fatalf("getPublicReposWithAPIURL for user failed: %v", err)
		}
		if len(repos) != 1 || repos[0] != "https://github.com/testuser/repo1.git" {
			t.Errorf("unexpected user repos: %v", repos)
		}
	})

	t.Run("Org Repos with Pagination", func(t *testing.T) {
		mockClient := mocks.NewMockClient(map[string]*http.Response{
			"https://api.github.com/users/testorg/repos": {
				StatusCode: http.StatusNotFound, // Trigger fallback to org
				Status:     "404 Not Found",
				Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
			},
			"https://api.github.com/orgs/testorg/repos": {
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}, "Link": []string{`<https://api.github.com/organizations/123/repos?page=2>; rel="next"`}},
				Body:       io.NopCloser(bytes.NewBufferString(`[{"clone_url": "https://github.com/testorg/repo1.git"}]`)),
			},
			"https://api.github.com/organizations/123/repos?page=2": {
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`[{"clone_url": "https://github.com/testorg/repo2.git"}]`)),
			},
		})
		client := setupMockClient(t, mockClient)
		repos, err := client.getPublicReposWithAPIURL(context.Background(), "https://api.github.com", "testorg")
		if err != nil {
			t.Fatalf("getPublicReposWithAPIURL for org failed: %v", err)
		}
		if len(repos) != 2 || repos[0] != "https://github.com/testorg/repo1.git" || repos[1] != "https://github.com/testorg/repo2.git" {
			t.Errorf("unexpected org repos: %v", repos)
		}
	})
}

func TestGetPublicRepos_Bad(t *testing.T) {
	t.Run("Not Found", func(t *testing.T) {
		mockClient := mocks.NewMockClient(map[string]*http.Response{
			"https://api.github.com/users/testuser/repos": {
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Not Found"}`)),
			},
			"https://api.github.com/orgs/testuser/repos": {
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Not Found"}`)),
			},
		})
		client := setupMockClient(t, mockClient)
		_, err := client.getPublicReposWithAPIURL(context.Background(), "https://api.github.com", "testuser")
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		if !strings.Contains(err.Error(), "404 Not Found") {
			t.Errorf("expected '404 Not Found' in error message, got %q", err)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		mockClient := mocks.NewMockClient(map[string]*http.Response{
			"https://api.github.com/users/badjson/repos": {
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`[{"clone_url": "invalid}`)),
			},
		})
		client := setupMockClient(t, mockClient)
		_, err := client.getPublicReposWithAPIURL(context.Background(), "https://api.github.com", "badjson")
		if err == nil {
			t.Fatal("expected an error for invalid JSON, but got nil")
		}
	})
}

func TestGetPublicRepos_Ugly(t *testing.T) {
	t.Run("Empty Repo List", func(t *testing.T) {
		mockClient := mocks.NewMockClient(map[string]*http.Response{
			"https://api.github.com/users/empty/repos": {
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`[]`)),
			},
		})
		client := setupMockClient(t, mockClient)
		repos, err := client.getPublicReposWithAPIURL(context.Background(), "https://api.github.com", "empty")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repos) != 0 {
			t.Errorf("expected 0 repos, got %d", len(repos))
		}
	})
}

func TestFindNextURL_Good(t *testing.T) {
	client := &githubClient{}
	linkHeader := `<https://api.github.com/organizations/123/repos?page=2>; rel="next", <https://api.github.com/organizations/123/repos?page=1>; rel="prev"`
	nextURL := client.findNextURL(linkHeader)
	if nextURL != "https://api.github.com/organizations/123/repos?page=2" {
		t.Errorf("unexpected next URL: %s", nextURL)
	}
}

func TestFindNextURL_Bad(t *testing.T) {
	client := &githubClient{}
	linkHeader := `<https://api.github.com/organizations/123/repos?page=1>; rel="prev"`
	nextURL := client.findNextURL(linkHeader)
	if nextURL != "" {
		t.Errorf("unexpected next URL for header with no 'next': %s", nextURL)
	}

	nextURL = client.findNextURL("")
	if nextURL != "" {
		t.Errorf("unexpected next URL for empty header: %s", nextURL)
	}
}

func TestFindNextURL_Ugly(t *testing.T) {
	client := &githubClient{}
	// Malformed: missing angle brackets
	linkHeader := `https://api.github.com/organizations/123/repos?page=2; rel="next"`
	nextURL := client.findNextURL(linkHeader)
	if nextURL != "" {
		t.Errorf("unexpected next URL for malformed header: %s", nextURL)
	}
}

func TestNewAuthenticatedClient_Good(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	client := NewAuthenticatedClient(context.Background())
	if client == http.DefaultClient {
		t.Error("expected an authenticated client, but got http.DefaultClient")
	}
}

func TestNewAuthenticatedClient_Bad(t *testing.T) {
	// Unset the variable to ensure it's not present
	t.Setenv("GITHUB_TOKEN", "")
	client := NewAuthenticatedClient(context.Background())
	if client != http.DefaultClient {
		t.Error("expected http.DefaultClient when no token is set, but got something else")
	}
}

// setupMockClient is a helper function to inject a mock http.Client.
func setupMockClient(t *testing.T, mock *http.Client) *githubClient {
	client := &githubClient{}
	originalNewAuthenticatedClient := NewAuthenticatedClient
	NewAuthenticatedClient = func(ctx context.Context) *http.Client {
		return mock
	}
	// Restore the original function after the test
	t.Cleanup(func() {
		NewAuthenticatedClient = originalNewAuthenticatedClient
	})
	return client
}

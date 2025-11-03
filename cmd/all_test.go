package cmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/mocks"
)

func TestAllCmd_Good(t *testing.T) {
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
	rootCmd.AddCommand(allCmd)

	_, err := executeCommand(rootCmd, "all", "https://github.com/testuser", "--output", "/dev/null")
	if err != nil {
		t.Fatalf("all command failed: %v", err)
	}
}

func TestAllCmd_Bad(t *testing.T) {
	mockGithubClient := mocks.NewMockClient(map[string]*http.Response{
		"https://api.github.com/users/testuser/repos": {
			StatusCode: http.StatusNotFound,
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
	rootCmd.AddCommand(allCmd)

	_, err := executeCommand(rootCmd, "all", "https://github.com/testuser", "--output", "/dev/null")
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}
}

package mocks

import (
	"context"

	"github.com/Snider/Borg/pkg/github"
)

// MockGithubClient is a mock implementation of the GithubClient interface.
type MockGithubClient struct {
	PublicRepos []string
	Err         error
}

// NewMockGithubClient creates a new MockGithubClient.
func NewMockGithubClient(repos []string, err error) github.GithubClient {
	return &MockGithubClient{
		PublicRepos: repos,
		Err:         err,
	}
}

// GetPublicRepos mocks the retrieval of public repositories.
func (m *MockGithubClient) GetPublicRepos(ctx context.Context, owner string) ([]string, error) {
	return m.PublicRepos, m.Err
}

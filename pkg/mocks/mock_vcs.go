package mocks

import (
	"fmt"
	"io"

	"github.com/Snider/Borg/pkg/datanode"
)

// MockGitCloner is a mock implementation of the GitCloner interface.
type MockGitCloner struct {
	Responses map[string]struct {
		DN  *datanode.DataNode
		Err error
	}
}

// NewMockGitCloner creates a new MockGitCloner.
func NewMockGitCloner() *MockGitCloner {
	return &MockGitCloner{
		Responses: make(map[string]struct {
			DN  *datanode.DataNode
			Err error
		}),
	}
}

// AddResponse adds a mock response for a given repository URL.
func (m *MockGitCloner) AddResponse(repoURL string, dn *datanode.DataNode, err error) {
	m.Responses[repoURL] = struct {
		DN  *datanode.DataNode
		Err error
	}{DN: dn, Err: err}
}

// CloneGitRepository mocks the cloning of a Git repository.
func (m *MockGitCloner) CloneGitRepository(repoURL string, progress io.Writer) (*datanode.DataNode, error) {
	if resp, ok := m.Responses[repoURL]; ok {
		return resp.DN, resp.Err
	}
	return nil, fmt.Errorf("no mock response for %s", repoURL)
}

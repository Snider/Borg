package mocks

import (
	"io"

	"forge.lthn.ai/Snider/Borg/pkg/datanode"
	"forge.lthn.ai/Snider/Borg/pkg/vcs"
)

// MockGitCloner is a mock implementation of the GitCloner interface.
type MockGitCloner struct {
	DN  *datanode.DataNode
	Err error
}

// NewMockGitCloner creates a new MockGitCloner.
func NewMockGitCloner(dn *datanode.DataNode, err error) vcs.GitCloner {
	return &MockGitCloner{
		DN:  dn,
		Err: err,
	}
}

// CloneGitRepository mocks the cloning of a Git repository.
func (m *MockGitCloner) CloneGitRepository(repoURL string, progress io.Writer) (*datanode.DataNode, error) {
	return m.DN, m.Err
}

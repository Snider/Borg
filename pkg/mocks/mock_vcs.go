package mocks

import (
	"io"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/vcs"
)

// MockGitCloner is a mock implementation of the vcs.GitCloner interface, used
// for testing code that clones Git repositories. It allows setting a predefined
// DataNode and an error to be returned.
type MockGitCloner struct {
	// DN is the DataNode to be returned by CloneGitRepository.
	DN *datanode.DataNode
	// Err is the error to be returned by CloneGitRepository.
	Err error
}

// NewMockGitCloner creates a new MockGitCloner with the given DataNode and
// error. This is a convenience function for creating a mock Git cloner for
// tests.
//
// Example:
//
//	mockDN := datanode.New()
//	mockDN.AddData("file.txt", []byte("hello"))
//	mockCloner := mocks.NewMockGitCloner(mockDN, nil)
//	// use mockCloner in tests
func NewMockGitCloner(dn *datanode.DataNode, err error) vcs.GitCloner {
	return &MockGitCloner{
		DN:  dn,
		Err: err,
	}
}

// CloneGitRepository is the mock implementation of the vcs.GitCloner interface.
// It returns the pre-configured DataNode and error.
func (m *MockGitCloner) CloneGitRepository(repoURL string, progress io.Writer) (*datanode.DataNode, error) {
	return m.DN, m.Err
}

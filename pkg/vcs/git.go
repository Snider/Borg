// Package vcs provides functionality for interacting with version control
// systems, such as Git.
package vcs

import (
	"io"
	"os"
	"path/filepath"

	"github.com/Snider/Borg/pkg/datanode"

	"github.com/go-git/go-git/v5"
)

// GitCloner defines the interface for cloning Git repositories. This allows for
// mocking the cloner in tests.
type GitCloner interface {
	// CloneGitRepository clones a Git repository from a URL and packages its
	// contents into a DataNode.
	CloneGitRepository(repoURL string, progress io.Writer) (*datanode.DataNode, error)
}

// NewGitCloner creates and returns a new GitCloner.
//
// Example:
//
//	cloner := vcs.NewGitCloner()
//	dn, err := cloner.CloneGitRepository("https://github.com/example/repo.git", nil)
//	if err != nil {
//		// handle error
//	}
func NewGitCloner() GitCloner {
	return &gitCloner{}
}

type gitCloner struct{}

// CloneGitRepository clones a Git repository from a URL into a temporary
// directory, then packages the contents of the repository into a DataNode.
// The .git directory is excluded from the DataNode. If the repository is empty,
// an empty DataNode is returned.
func (g *gitCloner) CloneGitRepository(repoURL string, progress io.Writer) (*datanode.DataNode, error) {
	tempPath, err := os.MkdirTemp("", "borg-clone-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempPath)

	cloneOptions := &git.CloneOptions{
		URL: repoURL,
	}
	if progress != nil {
		cloneOptions.Progress = progress
	}

	_, err = git.PlainClone(tempPath, false, cloneOptions)
	if err != nil {
		if err.Error() == "remote repository is empty" {
			return datanode.New(), nil
		}
		return nil, err
	}

	dn := datanode.New()
	err = filepath.Walk(tempPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip the .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(tempPath, path)
			if err != nil {
				return err
			}
			dn.AddData(relPath, content)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return dn, nil
}

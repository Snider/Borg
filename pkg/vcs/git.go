package vcs

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Snider/Borg/pkg/datanode"

	"github.com/go-git/go-git/v5"
)

// GitCloner is an interface for cloning Git repositories.
type GitCloner interface {
	CloneGitRepository(repoURL string, progress io.Writer) (*datanode.DataNode, error)
}

// NewGitCloner creates a new GitCloner.
func NewGitCloner() GitCloner {
	return &gitCloner{}
}

type gitCloner struct{}

// CloneGitRepository clones a Git repository from a URL and packages it into a DataNode.
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

	var lastErr error
	retries := 3
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for i := 0; i < retries; i++ {
		_, err = git.PlainClone(tempPath, false, cloneOptions)
		if err == nil {
			lastErr = nil
			break
		}

		lastErr = err

		// Handle non-retryable error
		if err.Error() == "remote repository is empty" {
			return datanode.New(), nil
		}

		// Don't wait on the last attempt
		if i < retries-1 {
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	if lastErr != nil {
		return nil, lastErr
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

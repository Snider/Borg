package vcs

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/Snider/Borg/pkg/datanode"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// GitCloner is an interface for cloning Git repositories.
type GitCloner interface {
	CloneGitRepository(repoURL string, progress io.Writer) (*datanode.DataNode, error)
}

// NewGitCloner creates a new GitCloner with the default http client.
func NewGitCloner() GitCloner {
	return NewGitClonerWithClient(http.DefaultClient)
}

// NewGitClonerWithClient creates a new GitCloner with a custom http.Client.
func NewGitClonerWithClient(client *http.Client) GitCloner {
	if client == nil {
		client = http.DefaultClient
	}
	return &gitCloner{
		httpClient: client,
	}
}

type gitCloner struct {
	httpClient *http.Client
}

var cloneMutex = &sync.Mutex{}

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

	cloneMutex.Lock()
	originalClient := githttp.DefaultClient
	githttp.DefaultClient = githttp.NewClient(g.httpClient)
	defer func() {
		githttp.DefaultClient = originalClient
		cloneMutex.Unlock()
	}()

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

package vcs

import (
	"io"
	"os"
	"path/filepath"

	"github.com/Snider/Borg/pkg/datanode"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

// GitCloneOptions defines the options for cloning a Git repository.
type GitCloneOptions struct {
	Depth       int
	AllBranches bool
	AllTags     bool
	FullHistory bool
}

// GitCloner is an interface for cloning Git repositories.
type GitCloner interface {
	CloneGitRepository(repoURL string, options GitCloneOptions, progress io.Writer) (*datanode.DataNode, error)
}

// NewGitCloner creates a new GitCloner.
func NewGitCloner() GitCloner {
	return &gitCloner{}
}

type gitCloner struct{}

// CloneGitRepository clones a Git repository from a URL and packages it into a DataNode.
func (g *gitCloner) CloneGitRepository(repoURL string, options GitCloneOptions, progress io.Writer) (*datanode.DataNode, error) {
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

	if options.Depth > 0 {
		cloneOptions.Depth = options.Depth
	}

	if options.AllTags {
		cloneOptions.Tags = git.AllTags
	}

	repo, err := git.PlainClone(tempPath, false, cloneOptions)
	if err != nil {
		if err.Error() == "remote repository is empty" {
			return datanode.New(), nil
		}
		return nil, err
	}

	if options.AllBranches {
		remote, err := repo.Remote("origin")
		if err != nil {
			return nil, err
		}

		err = remote.Fetch(&git.FetchOptions{
			RefSpecs: []config.RefSpec{"+refs/heads/*:refs/remotes/origin/*"},
			Progress: progress,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return nil, err
		}
	}

	dn := datanode.New()
	err = filepath.Walk(tempPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip the .git directory if we are not preserving history
		if !options.FullHistory && info.IsDir() && info.Name() == ".git" {
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

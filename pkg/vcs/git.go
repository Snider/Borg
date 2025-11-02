package vcs

import (
	"io"
	"os"
	"path/filepath"

	"github.com/Snider/Borg/pkg/datanode"

	"github.com/go-git/go-git/v5"
)

// CloneGitRepository clones a Git repository from a URL and packages it into a DataNode.
func CloneGitRepository(repoURL string, progress io.Writer) (*datanode.DataNode, error) {
	tempPath, err := os.MkdirTemp("", "borg-clone-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempPath)

	_, err = git.PlainClone(tempPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: progress,
	})
	if err != nil {
		return nil, err
	}

	dn := datanode.New()
	err = filepath.Walk(tempPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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

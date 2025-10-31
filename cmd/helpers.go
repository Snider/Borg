package cmd

import (
	"os"
	"path/filepath"

	"borg-data-collector/pkg/trix"

	"github.com/go-git/go-git/v5"
)

func addRepoToCube(repoURL string, cube *trix.Cube, clonePath string) error {
	_, err := git.PlainClone(clonePath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})

	if err != nil {
		return err
	}

	err = filepath.Walk(clonePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(clonePath, path)
			if err != nil {
				return err
			}
			cube.AddFile(relPath, content)
		}
		return nil
	})

	return err
}

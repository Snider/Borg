package diff

import (
	"bytes"
	"io"
	"io/fs"

	"github.com/Snider/Borg/pkg/datanode"
)

// Diff represents the differences between two DataNodes.
type Diff struct {
	Added    []string
	Removed  []string
	Modified []string
}

// fileInfo stores content for comparison.
type fileInfo struct {
	content []byte
}

// Compare compares two DataNodes and returns a Diff object.
func Compare(a, b *datanode.DataNode) (*Diff, error) {
	diff := &Diff{}
	filesA := make(map[string]fileInfo)
	filesB := make(map[string]fileInfo)

	// Walk through the first DataNode and collect file data
	err := a.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			file, err := a.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				return err
			}
			filesA[path] = fileInfo{content: content}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Walk through the second DataNode and collect file data
	err = b.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			file, err := b.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				return err
			}
			filesB[path] = fileInfo{content: content}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Find removed and modified files
	for path, infoA := range filesA {
		infoB, ok := filesB[path]
		if !ok {
			diff.Removed = append(diff.Removed, path)
		} else if !bytes.Equal(infoA.content, infoB.content) {
			diff.Modified = append(diff.Modified, path)
		}
	}

	// Find added files
	for path := range filesB {
		if _, ok := filesA[path]; !ok {
			diff.Added = append(diff.Added, path)
		}
	}

	return diff, nil
}

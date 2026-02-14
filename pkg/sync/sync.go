package sync

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"

	"github.com/Snider/Borg/pkg/datanode"
)

// SyncStrategy defines the strategy for a sync operation.
type SyncStrategy string

const (
	// AppendStrategy adds new files only.
	AppendStrategy SyncStrategy = "append"
	// MirrorStrategy matches the source exactly.
	MirrorStrategy SyncStrategy = "mirror"
	// UpdateStrategy updates existing files and adds new ones.
	UpdateStrategy SyncStrategy = "update"
)

// Sync merges two DataNodes based on a given strategy.
func Sync(a, b *datanode.DataNode, strategy SyncStrategy) (*datanode.DataNode, error) {
	result := datanode.New()
	filesA := make(map[string][]byte)
	filesB := make(map[string][]byte)

	// Helper function to walk a DataNode and populate a map
	walkAndCollect := func(dn *datanode.DataNode, fileMap map[string][]byte) error {
		return dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				file, err := dn.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				content, err := io.ReadAll(file)
				if err != nil {
					return err
				}
				fileMap[path] = content
			}
			return nil
		})
	}

	if err := walkAndCollect(a, filesA); err != nil {
		return nil, fmt.Errorf("failed to walk source datanode: %w", err)
	}
	if err := walkAndCollect(b, filesB); err != nil {
		return nil, fmt.Errorf("failed to walk target datanode: %w", err)
	}

	switch strategy {
	case AppendStrategy:
		// Add all files from A first
		for path, content := range filesA {
			result.AddData(path, content)
		}
		// Add files from B that are not in A
		for path, content := range filesB {
			if _, exists := filesA[path]; !exists {
				result.AddData(path, content)
			}
		}
	case MirrorStrategy:
		// Result is an exact copy of B
		for path, content := range filesB {
			result.AddData(path, content)
		}
	case UpdateStrategy:
		// Add all files from A first
		for path, content := range filesA {
			result.AddData(path, content)
		}
		// Add or update files from B
		for path, contentB := range filesB {
			contentA, exists := filesA[path]
			if !exists || !bytes.Equal(contentA, contentB) {
				result.AddData(path, contentB)
			}
		}
	default:
		return nil, fmt.Errorf("unknown sync strategy: %s", strategy)
	}

	return result, nil
}

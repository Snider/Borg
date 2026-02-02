package cmd

import (
	"bytes"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/storage"
)

func TestPullCmd(t *testing.T) {
	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "testfile.txt")

	// Mock the storage backend
	storage.NewStorage = func(u *url.URL) (storage.Storage, error) {
		return &MockStorage{
			ReadFunc: func(path string) (io.ReadCloser, error) {
				if path != "/remote/path" {
					t.Errorf("expected path '/remote/path', got '%s'", path)
				}
				return io.NopCloser(bytes.NewReader([]byte("pull test"))), nil
			},
		}, nil
	}

	// Execute the pull command
	root := NewRootCmd()
	root.AddCommand(NewPullCmd())
	_, err := executeCommand(root, "pull", "mock://bucket/remote/path", localPath)
	if err != nil {
		t.Fatalf("pull command failed: %v", err)
	}

	// Assertions
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("failed to read pulled file: %v", err)
	}
	if string(data) != "pull test" {
		t.Errorf("expected data 'pull test', got '%s'", string(data))
	}
}

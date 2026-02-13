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

func TestPushCmd(t *testing.T) {
	// Create a temporary file to push
	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(localPath, []byte("push test"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var writtenPath string
	var writtenData []byte

	// Mock the storage backend
	storage.NewStorage = func(u *url.URL) (storage.Storage, error) {
		return &MockStorage{
			WriteFunc: func(path string, data io.Reader) error {
				writtenPath = path
				writtenData, _ = io.ReadAll(data)
				return nil
			},
		}, nil
	}

	// Execute the push command
	root := NewRootCmd()
	root.AddCommand(NewPushCmd())
	_, err = executeCommand(root, "push", localPath, "mock://bucket/remote/path")
	if err != nil {
		t.Fatalf("push command failed: %v", err)
	}

	// Assertions
	if writtenPath != "/remote/path" {
		t.Errorf("expected path '/remote/path', got '%s'", writtenPath)
	}
	if string(writtenData) != "push test" {
		t.Errorf("expected data 'push test', got '%s'", string(writtenData))
	}
}

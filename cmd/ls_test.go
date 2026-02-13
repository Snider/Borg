package cmd

import (
	"net/url"
	"testing"

	"github.com/Snider/Borg/pkg/storage"
)

func TestLsCmd(t *testing.T) {
	// Mock the storage backend
	storage.NewStorage = func(u *url.URL) (storage.Storage, error) {
		return &MockStorage{
			ListFunc: func(path string) ([]string, error) {
				if path != "/remote/path" {
					t.Errorf("expected path '/remote/path', got '%s'", path)
				}
				return []string{"file1.txt", "file2.txt"}, nil
			},
		}, nil
	}

	// Execute the ls command
	root := NewRootCmd()
	root.AddCommand(NewLsCmd())
	output, err := executeCommand(root, "ls", "mock://bucket/remote/path")
	if err != nil {
		t.Fatalf("ls command failed: %v", err)
	}

	// Assertions
	expectedOutput := "file1.txt\nfile2.txt\n"
	if output != expectedOutput {
		t.Errorf("expected output '%s', got '%s'", expectedOutput, output)
	}
}

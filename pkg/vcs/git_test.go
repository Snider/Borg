package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCloneGitRepository(t *testing.T) {
	// Create a temporary directory for the bare repository
	bareRepoPath, err := os.MkdirTemp("", "bare-repo-")
	if err != nil {
		t.Fatalf("Failed to create temp dir for bare repo: %v", err)
	}
	defer os.RemoveAll(bareRepoPath)

	// Initialize a bare git repository
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = bareRepoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	// Clone the bare repository to a temporary directory to add a commit
	clonePath, err := os.MkdirTemp("", "clone-")
	if err != nil {
		t.Fatalf("Failed to create temp dir for clone: %v", err)
	}
	defer os.RemoveAll(clonePath)

	cmd = exec.Command("git", "clone", bareRepoPath, clonePath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to clone bare repo: %v", err)
	}

	// Create a file and commit it
	filePath := filepath.Join(clonePath, "foo.txt")
	if err := os.WriteFile(filePath, []byte("foo"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	cmd = exec.Command("git", "add", "foo.txt")
	cmd.Dir = clonePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = clonePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}
	cmd = exec.Command("git", "push", "origin", "master")
	cmd.Dir = clonePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git push: %v", err)
	}

	// Clone the repository using the function we're testing
	dn, err := CloneGitRepository("file://" + bareRepoPath)
	if err != nil {
		t.Fatalf("CloneGitRepository failed: %v", err)
	}

	// Verify the DataNode contains the correct file
	exists, err := dn.Exists("foo.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Errorf("Expected to find file foo.txt in DataNode, but it was not found")
	}
}

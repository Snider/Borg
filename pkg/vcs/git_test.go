package vcs

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a bare git repository with a single commit.
func setupTestRepo(t *testing.T) (repoPath string) {
	t.Helper()

	// Create a temporary directory for the bare repository.
	bareRepoPath, err := os.MkdirTemp("", "bare-repo-")
	if err != nil {
		t.Fatalf("Failed to create temp dir for bare repo: %v", err)
	}

	// Initialize the bare git repository.
	runCmd(t, bareRepoPath, "git", "init", "--bare")

	// Clone the bare repository to a temporary directory to add a commit.
	clonePath, err := os.MkdirTemp("", "clone-")
	if err != nil {
		t.Fatalf("Failed to create temp dir for clone: %v", err)
	}
	defer os.RemoveAll(clonePath)

	runCmd(t, clonePath, "git", "clone", bareRepoPath, ".")

	// Create a file and commit it.
	filePath := filepath.Join(clonePath, "foo.txt")
	if err := os.WriteFile(filePath, []byte("foo"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	runCmd(t, clonePath, "git", "add", "foo.txt")
	runCmd(t, clonePath, "git", "config", "user.email", "test@example.com")
	runCmd(t, clonePath, "git", "config", "user.name", "Test User")
	runCmd(t, clonePath, "git", "commit", "-m", "Initial commit")
	runCmd(t, clonePath, "git", "push", "origin", "master")

	return bareRepoPath
}

// runCmd executes a command and fails the test if it fails.
func runCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if testing.Verbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Command %q failed: %v", strings.Join(append([]string{name}, args...), " "), err)
	}
}

func TestCloneGitRepository_Good(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer os.RemoveAll(repoPath)

	cloner := NewGitCloner()
	var out bytes.Buffer
	dn, err := cloner.CloneGitRepository("file://"+repoPath, &out)
	if err != nil {
		t.Fatalf("CloneGitRepository failed: %v\nOutput: %s", err, out.String())
	}

	// Verify the DataNode contains the correct file.
	exists, err := dn.Exists("foo.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Errorf("Expected to find file foo.txt in DataNode, but it was not found")
	}
}

func TestCloneGitRepository_Bad(t *testing.T) {
	t.Run("Non-existent repository", func(t *testing.T) {
		cloner := NewGitCloner()
		_, err := cloner.CloneGitRepository("file:///non-existent-repo", io.Discard)
		if err == nil {
			t.Fatal("Expected an error for a non-existent repository, but got nil")
		}
		if !strings.Contains(err.Error(), "repository not found") {
			t.Errorf("Expected error to be about 'repository not found', but got: %v", err)
		}
	})

	t.Run("Invalid URL", func(t *testing.T) {
		cloner := NewGitCloner()
		_, err := cloner.CloneGitRepository("not-a-valid-url", io.Discard)
		if err == nil {
			t.Fatal("Expected an error for an invalid URL, but got nil")
		}
	})
}

func TestCloneGitRepository_Ugly(t *testing.T) {
	t.Run("Empty repository", func(t *testing.T) {
		bareRepoPath, err := os.MkdirTemp("", "empty-bare-repo-")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(bareRepoPath)
		runCmd(t, bareRepoPath, "git", "init", "--bare")

		cloner := NewGitCloner()
		dn, err := cloner.CloneGitRepository("file://"+bareRepoPath, io.Discard)
		if err != nil {
			t.Fatalf("CloneGitRepository failed on empty repo: %v", err)
		}
		if dn == nil {
			t.Fatal("Expected a non-nil datanode for an empty repo")
		}
		// You might want to check if the datanode is empty, but for now, just checking for no error is enough.
	})
}

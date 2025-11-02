package cmd

import (
	"os"
	"os/exec"
	"testing"
)

func TestMain(t *testing.T) {
	// This is a bit of a hack, but it's the easiest way to test the main function.
	// We're just making sure that the application doesn't crash when it's run.
	Execute()
}

func TestE2E(t *testing.T) {
	taskPath, err := findTaskExecutable()
	if err != nil {
		t.Fatalf("Failed to find task executable: %v", err)
	}
	cmd := exec.Command(taskPath, "test-e2e")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run e2e test: %v\n%s", err, output)
	}
}

func findTaskExecutable() (string, error) {
	// First, try to find "task" in the system's PATH
	path, err := exec.LookPath("task")
	if err == nil {
		return path, nil
	}

	// If not found in PATH, try to find it in the user's Go bin directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	goBin := home + "/go/bin/task"
	if _, err := os.Stat(goBin); err == nil {
		return goBin, nil
	}

	return "", os.ErrNotExist
}

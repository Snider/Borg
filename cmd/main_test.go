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
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}
	taskPath := home + "/go/bin/task"
	cmd := exec.Command(taskPath, "test-e2e")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run e2e test: %v\n%s", err, output)
	}
}

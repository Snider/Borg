package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectDockerCmd_Good(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		t.Cleanup(func() {
			// Reset the command's state after the test.
			RootCmd.SetArgs([]string{})
		})
		// Use a small, public image for testing
		imageRef := "hello-world"

		// Create a temporary directory to store the output
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "hello-world.tar")

		// Execute the command
		output, err := executeCommand(RootCmd, "collect", "docker", imageRef, "--output", outputFile)
		if err != nil {
			t.Fatalf("executeCommand() returned an unexpected error: %v, output: %s", err, output)
		}

		// Check if the output file was created
		fileInfo, err := os.Stat(outputFile)
		if os.IsNotExist(err) {
			t.Fatalf("collect docker command did not create the output file: %s", outputFile)
		}
		if err != nil {
			t.Fatalf("error stating output file: %v", err)
		}
	})

	t.Run("Platform", func(t *testing.T) {
		t.Cleanup(func() {
			// Reset the command's state after the test.
			RootCmd.SetArgs([]string{})
		})
		// Use a multi-platform image for testing
		imageRef := "nginx"
		platform := "linux/arm64"

		// Create a temporary directory to store the output
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "nginx.tar")

		// Execute the command
		output, err := executeCommand(RootCmd, "collect", "docker", imageRef, "--output", outputFile, "--platform", platform)
		if err != nil {
			t.Fatalf("executeCommand() returned an unexpected error: %v, output: %s", err, output)
		}

		// Check if the output file was created
		fileInfo, err := os.Stat(outputFile)
		if os.IsNotExist(err) {
			t.Fatalf("collect docker command did not create the output file: %s", outputFile)
		}
		if err != nil {
			t.Fatalf("error stating output file: %v", err)
		}

		// Check if the file is not empty
		if fileInfo.Size() == 0 {
			t.Errorf("the created output file is empty")
		}

		// Check for the success message in the output
		expectedOutput := "Docker image saved to"
		if !strings.Contains(output, expectedOutput) {
			t.Errorf("expected output to contain %q, but got %q", expectedOutput, output)
		}
	})
}

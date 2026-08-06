package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollect(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		// Use a small, public image for testing
		imageRef := "hello-world"

		// Create a temporary directory to store the output
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "hello-world.tar")

		// Call the Collect function
		err := Collect(imageRef, outputFile, false, "", "")
		if err != nil {
			t.Fatalf("Collect() returned an unexpected error: %v", err)
		}

		// Check if the output file was created
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Fatalf("Collect() did not create the output file: %s", outputFile)
		}
	})

	t.Run("Platform", func(t *testing.T) {
		// Use a multi-platform image for testing
		imageRef := "nginx"
		platform := "linux/arm64"

		// Create a temporary directory to store the output
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "nginx.tar")

		// Call the Collect function
		err := Collect(imageRef, outputFile, false, platform, "")
		if err != nil {
			t.Fatalf("Collect() returned an unexpected error: %v", err)
		}

		// Check if the output file was created
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Fatalf("Collect() did not create the output file: %s", outputFile)
		}
	})
}

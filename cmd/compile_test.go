package cmd

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCompileCmd_Good(t *testing.T) {
	tempDir := t.TempDir()
	borgfilePath := filepath.Join(tempDir, "Borgfile")
	outputMatrixPath := filepath.Join(tempDir, "test.matrix")
	fileToAddPath := filepath.Join(tempDir, "test.txt")

	// Create a dummy file to add to the matrix.
	err := os.WriteFile(fileToAddPath, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a dummy Borgfile.
	borgfileContent := "ADD " + fileToAddPath + " /test.txt"
	err = os.WriteFile(borgfilePath, []byte(borgfileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create Borgfile: %v", err)
	}

	// Run the compile command.
	rootCmd := NewRootCmd()
	rootCmd.AddCommand(compileCmd)
	_, err = executeCommand(rootCmd, "compile", "-f", borgfilePath, "-o", outputMatrixPath)
	if err != nil {
		t.Fatalf("compile command failed: %v", err)
	}

	// Verify the output matrix file.
	matrixFile, err := os.Open(outputMatrixPath)
	if err != nil {
		t.Fatalf("failed to open output matrix file: %v", err)
	}
	defer matrixFile.Close()

	tr := tar.NewReader(matrixFile)
	foundConfig := false
	foundRootFS := false
	foundTestFile := false
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read tar header: %v", err)
		}

		switch header.Name {
		case "config.json":
			foundConfig = true
		case "rootfs/":
			foundRootFS = true
		case "rootfs/test.txt":
			foundTestFile = true
		}
	}

	if !foundConfig {
		t.Error("config.json not found in matrix")
	}
	if !foundRootFS {
		t.Error("rootfs/ not found in matrix")
	}
	if !foundTestFile {
		t.Error("rootfs/test.txt not found in matrix")
	}
}

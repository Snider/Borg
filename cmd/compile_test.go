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
	rootCmd.AddCommand(GetCompileCmd())
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
	found := make(map[string]bool)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read tar header: %v", err)
		}
		found[header.Name] = true
	}

	expectedFiles := []string{"config.json", "rootfs/", "rootfs/test.txt"}
	for _, f := range expectedFiles {
		if !found[f] {
			t.Errorf("%s not found in matrix tarball", f)
		}
	}
}

func TestCompileCmd_Bad(t *testing.T) {
	t.Run("Invalid Borgfile instruction", func(t *testing.T) {
		tempDir := t.TempDir()
		borgfilePath := filepath.Join(tempDir, "Borgfile")
		outputMatrixPath := filepath.Join(tempDir, "test.matrix")

		// Create a dummy Borgfile with an invalid instruction.
		borgfileContent := "INVALID_INSTRUCTION"
		err := os.WriteFile(borgfilePath, []byte(borgfileContent), 0644)
		if err != nil {
			t.Fatalf("failed to create Borgfile: %v", err)
		}

		// Run the compile command.
		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetCompileCmd())
		_, err = executeCommand(rootCmd, "compile", "-f", borgfilePath, "-o", outputMatrixPath)
		if err == nil {
			t.Fatal("compile command should have failed but did not")
		}
	})

	t.Run("Missing input file", func(t *testing.T) {
		tempDir := t.TempDir()
		borgfilePath := filepath.Join(tempDir, "Borgfile")
		outputMatrixPath := filepath.Join(tempDir, "test.matrix")

		// Create a dummy Borgfile that references a non-existent file.
		borgfileContent := "ADD /non/existent/file /test.txt"
		err := os.WriteFile(borgfilePath, []byte(borgfileContent), 0644)
		if err != nil {
			t.Fatalf("failed to create Borgfile: %v", err)
		}

		// Run the compile command.
		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetCompileCmd())
		_, err = executeCommand(rootCmd, "compile", "-f", borgfilePath, "-o", outputMatrixPath)
		if err == nil {
			t.Fatal("compile command should have failed but did not")
		}
	})
}

func TestCompileCmd_Ugly(t *testing.T) {
	t.Run("Empty Borgfile", func(t *testing.T) {
		tempDir := t.TempDir()
		borgfilePath := filepath.Join(tempDir, "Borgfile")
		outputMatrixPath := filepath.Join(tempDir, "test.matrix")

		// Create an empty Borgfile.
		err := os.WriteFile(borgfilePath, []byte(""), 0644)
		if err != nil {
			t.Fatalf("failed to create Borgfile: %v", err)
		}

		// Run the compile command.
		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetCompileCmd())
		_, err = executeCommand(rootCmd, "compile", "-f", borgfilePath, "-o", outputMatrixPath)
		if err != nil {
			t.Fatalf("compile command failed for empty Borgfile: %v", err)
		}
	})
}

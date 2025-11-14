package cmd

import (
	"archive/tar"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/matrix"
)

func helperProcess(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Exit(0)
}

func TestRunCmd_Good(t *testing.T) {
	// Create a dummy matrix file.
	matrixPath := createDummyMatrix(t)

	// Mock the exec.Command function.
	origExecCommand := execCommand
	execCommand = helperProcess
	t.Cleanup(func() {
		execCommand = origExecCommand
	})

	// Run the run command.
	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetRunCmd())
	_, err := executeCommand(rootCmd, "run", matrixPath)
	if err != nil {
		t.Fatalf("run command failed: %v", err)
	}
}

func TestRunCmd_Bad(t *testing.T) {
	t.Run("Missing input file", func(t *testing.T) {
		// Run the run command with a non-existent file.
		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetRunCmd())
		_, err := executeCommand(rootCmd, "run", "/non/existent/file.matrix")
		if err == nil {
			t.Fatal("run command should have failed but did not")
		}
	})
}

func TestRunCmd_Ugly(t *testing.T) {
	t.Run("Invalid matrix file", func(t *testing.T) {
		// Create an invalid (non-tar) matrix file.
		tempDir := t.TempDir()
		matrixPath := filepath.Join(tempDir, "invalid.matrix")
		err := os.WriteFile(matrixPath, []byte("this is not a tar file"), 0644)
		if err != nil {
			t.Fatalf("failed to create invalid matrix file: %v", err)
		}

		// Run the run command.
		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetRunCmd())
		_, err = executeCommand(rootCmd, "run", matrixPath)
		if err == nil {
			t.Fatal("run command should have failed but did not")
		}
	})
}

// createDummyMatrix creates a valid, empty matrix file for testing.
func createDummyMatrix(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	matrixPath := filepath.Join(tempDir, "test.matrix")
	matrixFile, err := os.Create(matrixPath)
	if err != nil {
		t.Fatalf("failed to create dummy matrix file: %v", err)
	}
	defer matrixFile.Close()

	tw := tar.NewWriter(matrixFile)

	// Add a dummy config.json.
	configContent := []byte(matrix.DefaultConfigJSON)
	hdr := &tar.Header{
		Name: "config.json",
		Mode: 0600,
		Size: int64(len(configContent)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	if _, err := tw.Write(configContent); err != nil {
		t.Fatalf("failed to write tar content: %v", err)
	}

	// Add the rootfs directory.
	hdr = &tar.Header{
		Name:     "rootfs/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	return matrixPath
}

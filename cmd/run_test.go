package cmd

import (
	"archive/tar"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/tim"
)

func TestRunCmd_Good(t *testing.T) {
	// Create a dummy tim file.
	timPath := createDummyTIM(t)

	// Mock the exec.Command function in the tim package.
	origExecCommand := tim.ExecCommand
	tim.ExecCommand = func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}
	t.Cleanup(func() {
		tim.ExecCommand = origExecCommand
	})

	// Run the run command.
	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetRunCmd())
	_, err := executeCommand(rootCmd, "run", timPath)
	if err != nil {
		t.Fatalf("run command failed: %v", err)
	}
}

func TestRunCmd_Bad(t *testing.T) {
	t.Run("Missing input file", func(t *testing.T) {
		// Run the run command with a non-existent file.
		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetRunCmd())
		_, err := executeCommand(rootCmd, "run", "/non/existent/file.tim")
		if err == nil {
			t.Fatal("run command should have failed but did not")
		}
	})
}

func TestRunCmd_Ugly(t *testing.T) {
	t.Run("Invalid tim file", func(t *testing.T) {
		// Create an invalid (non-tar) tim file.
		tempDir := t.TempDir()
		timPath := filepath.Join(tempDir, "invalid.tim")
		err := os.WriteFile(timPath, []byte("this is not a tar file"), 0644)
		if err != nil {
			t.Fatalf("failed to create invalid tim file: %v", err)
		}

		// Run the run command.
		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetRunCmd())
		_, err = executeCommand(rootCmd, "run", timPath)
		if err == nil {
			t.Fatal("run command should have failed but did not")
		}
	})
}

// createDummyTIM creates a valid, empty tim file for testing.
func createDummyTIM(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	timPath := filepath.Join(tempDir, "test.tim")
	timFile, err := os.Create(timPath)
	if err != nil {
		t.Fatalf("failed to create dummy tim file: %v", err)
	}
	defer timFile.Close()

	tw := tar.NewWriter(timFile)

	// Add a dummy config.json. This is not a valid config, but it's enough to test the run command.
	configContent := []byte(`{}`)
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
	return timPath
}

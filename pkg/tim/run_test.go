package tim

import (
	"archive/tar"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	// Create a dummy tim file.
	timPath := createDummyTim(t)

	// Mock the exec.Command function.
	origExecCommand := ExecCommand
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}
	t.Cleanup(func() {
		ExecCommand = origExecCommand
	})

	// Run the run command.
	err := Run(timPath)
	if err != nil {
		t.Fatalf("run command failed: %v", err)
	}
}

func TestRun_BadInput(t *testing.T) {
	// Test non-existent file
	err := Run("nonexistent.tim")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}

	// Test invalid file (not a tar)
	f, err := os.CreateTemp("", "bad-tim-*.tim")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Write([]byte("not a tar file"))
	f.Close()

	// Mock ExecCommand to fail if run called
	origExecCommand := ExecCommand
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		// If we reach here, it means we tried to run runc.
		// For a bad tar, we might still reach here.
		// Let's just return a command that fails.
		return exec.Command("false")
	}
	t.Cleanup(func() { ExecCommand = origExecCommand })

	err = Run(f.Name())
	if err == nil {
		t.Error("expected error when running bad tim file")
	}
}

func TestRun_ZipSlip(t *testing.T) {
	// Create a malicious tim file with ../ path
	file, err := os.CreateTemp("", "zipslip-*.tim")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	tw := tar.NewWriter(file)

	hdr := &tar.Header{
		Name: "../evil.txt",
		Mode: 0600,
		Size: 4,
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("evil")); err != nil {
		t.Fatal(err)
	}
	tw.Close()

	err = Run(file.Name())
	if err == nil {
		t.Fatal("expected error for zip slip attempt, got nil")
	}
	if !strings.Contains(err.Error(), "invalid file path") {
		t.Errorf("expected 'invalid file path' error, got: %v", err)
	}
}

// createDummyTim creates a valid, empty tim file for testing.
func createDummyTim(t *testing.T) string {
	t.Helper()
	// Create a dummy tim file.
	file, err := os.CreateTemp("", "tim-*.tim")
	if err != nil {
		t.Fatalf("failed to create dummy tim file: %v", err)
	}
	defer file.Close()

	tw := tar.NewWriter(file)

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
	return file.Name()
}

// TestHelperProcess isn't a real test. It's used as a helper for tests that need to mock exec.Command.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// The rest of the arguments are the command and its arguments.
	// In our case, we don't need to do anything with them.
	os.Exit(0)
}

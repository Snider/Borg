package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// mockExecCommand is used to mock the exec.Command function for testing.
func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess isn't a real test. It's used as a helper process
// for TestExtractMetadata. It simulates the behavior of the `pdfinfo` command.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command to mock!\n")
		os.Exit(1)
	}

	cmd, args := args[0], args[1:]
	if cmd == "pdfinfo" && len(args) == 1 {
		// Simulate pdfinfo output
		fmt.Println("Title:          Test Title")
		fmt.Println("Author:         Test Author 1,Test Author 2")
		fmt.Println("CreationDate:   Sun Jan  1 00:00:00 2023")
		fmt.Println("Pages:          42")
	}
}

func TestExtractMetadata(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	metadata, err := ExtractMetadata("dummy.pdf")
	if err != nil {
		t.Fatalf("ExtractMetadata failed: %v", err)
	}

	if metadata.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", metadata.Title)
	}
	if len(metadata.Authors) != 2 || metadata.Authors[0] != "Test Author 1" || metadata.Authors[1] != "Test Author 2" {
		t.Errorf("expected authors '[Test Author 1, Test Author 2]', got '%v'", metadata.Authors)
	}
	if metadata.Created != "Sun Jan  1 00:00:00 2023" {
		t.Errorf("expected creation date 'Sun Jan  1 00:00:00 2023', got '%s'", metadata.Created)
	}
	if metadata.Pages != 42 {
		t.Errorf("expected 42 pages, got %d", metadata.Pages)
	}
	if metadata.File != "dummy.pdf" {
		t.Errorf("expected file 'dummy.pdf', got '%s'", metadata.File)
	}
}

func TestExtractMetadata_CommandError(t *testing.T) {
	execCommand = func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess_Error", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}
	defer func() { execCommand = exec.Command }()

	_, err := ExtractMetadata("dummy.pdf")
	if err == nil {
		t.Fatal("expected an error from exec.Command, but got nil")
	}
	if !strings.Contains(err.Error(), "exit status 1") {
		t.Errorf("expected error to contain 'exit status 1', got '%v'", err)
	}
}

func TestHelperProcess_Error(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Simulate an error by writing to stderr and exiting with a non-zero status
	fmt.Fprintf(os.Stderr, "pdfinfo error")
	os.Exit(1)
}

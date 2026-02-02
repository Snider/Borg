package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestSearchCommand_WithoutIndex(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.dat")

	// Create a sample DataNode
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello world"))
	dn.AddData("file2.go", []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}"))
	tarData, err := dn.ToTar()
	if err != nil {
		t.Fatalf("failed to create tar: %v", err)
	}
	if err := os.WriteFile(archivePath, tarData, 0644); err != nil {
		t.Fatalf("failed to write archive: %v", err)
	}

	// Run the search command
	output, err := executeCommand(RootCmd, "search", archivePath, "hello")
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}

	if !strings.Contains(output, "file1.txt:1: hello world") {
		t.Errorf("expected to find 'hello' in file1.txt, got: %s", output)
	}
	if !strings.Contains(output, "file2.go:4: println(\"hello\")") {
		t.Errorf("expected to find 'hello' in file2.go, got: %s", output)
	}
}

func TestSearchCommand_WithIndex(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.dat")

	// Create a sample DataNode
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello world"))
	dn.AddData("file2.go", []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}"))
	tarData, err := dn.ToTar()
	if err != nil {
		t.Fatalf("failed to create tar: %v", err)
	}
	if err := os.WriteFile(archivePath, tarData, 0644); err != nil {
		t.Fatalf("failed to write archive: %v", err)
	}

	// Run the index command
	_, err = executeCommand(RootCmd, "index", archivePath)
	if err != nil {
		t.Fatalf("index command failed: %v", err)
	}

	// Run the search command
	output, err := executeCommand(RootCmd, "search", archivePath, "hello")
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}

	if !strings.Contains(output, "file1.txt:1: hello world") {
		t.Errorf("expected to find 'hello' in file1.txt, got: %s", output)
	}
	if !strings.Contains(output, "file2.go:4: println(\"hello\")") {
		t.Errorf("expected to find 'hello' in file2.go, got: %s", output)
	}
}

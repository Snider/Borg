package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestIndexCommand_Good(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.dat")

	// Create a sample DataNode
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello world"))
	dn.AddData("file2.go", []byte("package main"))
	tarData, err := dn.ToTar()
	if err != nil {
		t.Fatalf("failed to create tar: %v", err)
	}
	if err := os.WriteFile(archivePath, tarData, 0644); err != nil {
		t.Fatalf("failed to write archive: %v", err)
	}

	// Run the index command
	output, err := executeCommand(RootCmd, "index", archivePath)
	if err != nil {
		t.Fatalf("index command failed: %v", err)
	}

	if !strings.Contains(output, "Successfully built index") {
		t.Errorf("expected success message, got: %s", output)
	}

	// Verify that the index directory and files were created
	indexDir := filepath.Join(tmpDir, ".borg-index")
	if _, err := os.Stat(indexDir); os.IsNotExist(err) {
		t.Fatalf(".borg-index directory was not created")
	}

	filesJSONPath := filepath.Join(indexDir, "files.json")
	if _, err := os.Stat(filesJSONPath); os.IsNotExist(err) {
		t.Fatalf("files.json was not created")
	}

	trigramIdxPath := filepath.Join(indexDir, "trigram.idx")
	if _, err := os.Stat(trigramIdxPath); os.IsNotExist(err) {
		t.Fatalf("trigram.idx was not created")
	}
}

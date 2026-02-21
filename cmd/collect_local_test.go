package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectLocalStreaming_Good(t *testing.T) {
	// Create a temp directory with some test files
	srcDir := t.TempDir()
	outDir := t.TempDir()

	// Create files in subdirectories
	subDir := filepath.Join(srcDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	files := map[string]string{
		"hello.txt":        "hello world",
		"subdir/nested.go": "package main\n",
	}
	for name, content := range files {
		path := filepath.Join(srcDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	output := filepath.Join(outDir, "test.stim")
	err := CollectLocalStreaming(srcDir, output, "gz", "test-password")
	if err != nil {
		t.Fatalf("CollectLocalStreaming() error = %v", err)
	}

	// Verify file exists and is non-empty
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("output file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
}

func TestCollectLocalStreaming_Decrypt_Good(t *testing.T) {
	// Create a temp directory with known files
	srcDir := t.TempDir()
	outDir := t.TempDir()

	subDir := filepath.Join(srcDir, "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	expectedFiles := map[string]string{
		"README.md":    "# Test Project\n",
		"pkg/main.go":  "package main\n\nfunc main() {}\n",
	}
	for name, content := range expectedFiles {
		path := filepath.Join(srcDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	password := "decrypt-test-pw"
	output := filepath.Join(outDir, "roundtrip.stim")

	// Collect
	err := CollectLocalStreaming(srcDir, output, "gz", password)
	if err != nil {
		t.Fatalf("CollectLocalStreaming() error = %v", err)
	}

	// Decrypt
	dn, err := DecryptStimV2(output, password)
	if err != nil {
		t.Fatalf("DecryptStimV2() error = %v", err)
	}

	// Verify each expected file exists in the DataNode
	for name, wantContent := range expectedFiles {
		f, err := dn.Open(name)
		if err != nil {
			t.Errorf("file %q not found in DataNode: %v", name, err)
			continue
		}
		buf := make([]byte, 4096)
		n, _ := f.Read(buf)
		f.Close()
		got := string(buf[:n])
		if got != wantContent {
			t.Errorf("file %q content mismatch:\n  got:  %q\n  want: %q", name, got, wantContent)
		}
	}
}

func TestCollectLocalStreaming_BrokenSymlink_Good(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	// Create a regular file
	if err := os.WriteFile(filepath.Join(srcDir, "real.txt"), []byte("I exist"), 0644); err != nil {
		t.Fatalf("failed to write real.txt: %v", err)
	}

	// Create a broken symlink pointing to a nonexistent target
	brokenLink := filepath.Join(srcDir, "broken-link")
	if err := os.Symlink("/nonexistent/target/file", brokenLink); err != nil {
		t.Fatalf("failed to create broken symlink: %v", err)
	}

	output := filepath.Join(outDir, "symlink.stim")
	err := CollectLocalStreaming(srcDir, output, "none", "sym-password")
	if err != nil {
		t.Fatalf("CollectLocalStreaming() should skip broken symlinks, got error = %v", err)
	}

	// Verify output exists and is non-empty
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("output file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}

	// Decrypt and verify the broken symlink was skipped
	dn, err := DecryptStimV2(output, "sym-password")
	if err != nil {
		t.Fatalf("DecryptStimV2() error = %v", err)
	}

	// real.txt should be present
	if _, err := dn.Stat("real.txt"); err != nil {
		t.Error("expected real.txt in DataNode but it's missing")
	}

	// broken-link should NOT be present
	exists, _ := dn.Exists("broken-link")
	if exists {
		t.Error("broken symlink should have been skipped but was found in DataNode")
	}
}

func TestCollectLocalStreaming_Bad(t *testing.T) {
	outDir := t.TempDir()
	output := filepath.Join(outDir, "should-not-exist.stim")

	err := CollectLocalStreaming("/nonexistent/path/that/does/not/exist", output, "none", "password")
	if err == nil {
		t.Fatal("expected error for nonexistent directory, got nil")
	}

	// Verify no partial output file was left behind
	if _, statErr := os.Stat(output); statErr == nil {
		t.Error("partial output file should have been cleaned up")
	}
}

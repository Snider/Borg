package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestFullPipeline_Good exercises the complete streaming pipeline end-to-end
// with realistic directory contents including nested dirs, a large file that
// crosses the AEAD block boundary, valid and broken symlinks, and a hidden file.
// Each compression mode (none, gz, xz) is tested as a subtest.
func TestFullPipeline_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build a realistic source directory.
	srcDir := t.TempDir()

	// Regular files at root level.
	writeFile(t, srcDir, "readme.md", "# My Project\n\nA description.\n")
	writeFile(t, srcDir, "config.json", `{"version":"1.0","debug":false}`)

	// Nested directories with source code.
	mkdirAll(t, srcDir, "src")
	mkdirAll(t, srcDir, "src/pkg")
	writeFile(t, srcDir, "src/main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")
	writeFile(t, srcDir, "src/pkg/lib.go", "package pkg\n\n// Lib is a library function.\nfunc Lib() string { return \"lib\" }\n")

	// Large file: 1 MiB + 1 byte — crosses the 64 KiB block boundary used by
	// the chunked AEAD streaming encryption. Fill with a deterministic pattern
	// so we can verify content after round-trip.
	const largeSize = 1024*1024 + 1
	largeContent := make([]byte, largeSize)
	for i := range largeContent {
		largeContent[i] = byte(i % 251) // prime mod for non-trivial pattern
	}
	writeFileBytes(t, srcDir, "large.bin", largeContent)

	// Valid symlink pointing at a relative target.
	if err := os.Symlink("readme.md", filepath.Join(srcDir, "link-to-readme")); err != nil {
		t.Fatalf("failed to create valid symlink: %v", err)
	}

	// Broken symlink pointing at a nonexistent absolute path.
	if err := os.Symlink("/nonexistent/target", filepath.Join(srcDir, "broken-link")); err != nil {
		t.Fatalf("failed to create broken symlink: %v", err)
	}

	// Hidden file (dot-prefixed).
	writeFile(t, srcDir, ".hidden", "secret stuff\n")

	// Run each compression mode as a subtest.
	modes := []string{"none", "gz", "xz"}
	for _, comp := range modes {
		comp := comp // capture
		t.Run("compression="+comp, func(t *testing.T) {
			outDir := t.TempDir()
			outFile := filepath.Join(outDir, "pipeline-"+comp+".stim")
			password := "integration-test-pw-" + comp

			// Step 1: Collect (walk -> tar -> compress -> encrypt -> file).
			if err := CollectLocalStreaming(srcDir, outFile, comp, password); err != nil {
				t.Fatalf("CollectLocalStreaming(%q) error = %v", comp, err)
			}

			// Step 2: Verify output exists and is non-empty.
			info, err := os.Stat(outFile)
			if err != nil {
				t.Fatalf("output file does not exist: %v", err)
			}
			if info.Size() == 0 {
				t.Fatal("output file is empty")
			}

			// Step 3: Decrypt back into a DataNode.
			dn, err := DecryptStimV2(outFile, password)
			if err != nil {
				t.Fatalf("DecryptStimV2() error = %v", err)
			}

			// Step 4: Verify all regular files exist in the DataNode.
			expectedFiles := []string{
				"readme.md",
				"config.json",
				"src/main.go",
				"src/pkg/lib.go",
				"large.bin",
				".hidden",
			}
			for _, name := range expectedFiles {
				exists, eerr := dn.Exists(name)
				if eerr != nil {
					t.Errorf("Exists(%q) error = %v", name, eerr)
					continue
				}
				if !exists {
					t.Errorf("expected file %q in DataNode but it is missing", name)
				}
			}

			// Verify the valid symlink was included.
			linkExists, _ := dn.Exists("link-to-readme")
			if !linkExists {
				t.Error("expected symlink link-to-readme in DataNode but it is missing")
			}

			// Step 5: Verify large file has correct content (first byte check).
			f, err := dn.Open("large.bin")
			if err != nil {
				t.Fatalf("Open(large.bin) error = %v", err)
			}
			defer f.Close()

			// Read the entire large file and verify size and first byte.
			allData, err := io.ReadAll(f)
			if err != nil {
				t.Fatalf("reading large.bin: %v", err)
			}
			if len(allData) != largeSize {
				t.Errorf("large.bin size = %d, want %d", len(allData), largeSize)
			}
			if len(allData) > 0 && allData[0] != byte(0%251) {
				t.Errorf("large.bin first byte = %d, want %d", allData[0], byte(0%251))
			}

			// Verify content integrity of the whole large file.
			if !bytes.Equal(allData, largeContent) {
				t.Error("large.bin content does not match original after round-trip")
			}

			// Step 6: Verify broken symlink was skipped.
			brokenExists, _ := dn.Exists("broken-link")
			if brokenExists {
				t.Error("broken symlink should have been skipped but was found in DataNode")
			}
		})
	}
}

// TestFullPipeline_WrongPassword_Bad encrypts with one password and attempts
// to decrypt with a different password, verifying that an error is returned.
func TestFullPipeline_WrongPassword_Bad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srcDir := t.TempDir()
	outDir := t.TempDir()

	writeFile(t, srcDir, "secret.txt", "this is confidential\n")

	outFile := filepath.Join(outDir, "wrong-pw.stim")

	// Encrypt with the correct password.
	if err := CollectLocalStreaming(srcDir, outFile, "none", "correct-password"); err != nil {
		t.Fatalf("CollectLocalStreaming() error = %v", err)
	}

	// Attempt to decrypt with the wrong password.
	_, err := DecryptStimV2(outFile, "wrong-password")
	if err == nil {
		t.Fatal("expected error when decrypting with wrong password, got nil")
	}
}

// --- helpers ---

func writeFile(t *testing.T, base, rel, content string) {
	t.Helper()
	path := filepath.Join(base, rel)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", rel, err)
	}
}

func writeFileBytes(t *testing.T, base, rel string, data []byte) {
	t.Helper()
	path := filepath.Join(base, rel)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write %s: %v", rel, err)
	}
}

func mkdirAll(t *testing.T, base, rel string) {
	t.Helper()
	path := filepath.Join(base, rel)
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to mkdir %s: %v", rel, err)
	}
}

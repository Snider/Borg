package datanode

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAddPath_Good(t *testing.T) {
	// Create a temp directory with files and a nested subdirectory.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "world.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	dn := New()
	if err := dn.AddPath(dir, AddPathOptions{}); err != nil {
		t.Fatalf("AddPath failed: %v", err)
	}

	// Verify files are stored with paths relative to dir, using forward slashes.
	file, ok := dn.files["hello.txt"]
	if !ok {
		t.Fatal("hello.txt not found in datanode")
	}
	if string(file.content) != "hello" {
		t.Errorf("expected content 'hello', got %q", file.content)
	}

	file, ok = dn.files["sub/world.txt"]
	if !ok {
		t.Fatal("sub/world.txt not found in datanode")
	}
	if string(file.content) != "world" {
		t.Errorf("expected content 'world', got %q", file.content)
	}

	// Directories should not be stored explicitly.
	if _, ok := dn.files["sub"]; ok {
		t.Error("directories should not be stored as explicit entries")
	}
	if _, ok := dn.files["sub/"]; ok {
		t.Error("directories should not be stored as explicit entries")
	}
}

func TestAddPath_SkipBrokenSymlinks_Good(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks not reliably supported on Windows")
	}

	dir := t.TempDir()

	// Create a real file.
	if err := os.WriteFile(filepath.Join(dir, "real.txt"), []byte("real"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a broken symlink (target does not exist).
	if err := os.Symlink("/nonexistent/target", filepath.Join(dir, "broken.txt")); err != nil {
		t.Fatal(err)
	}

	dn := New()
	err := dn.AddPath(dir, AddPathOptions{SkipBrokenSymlinks: true})
	if err != nil {
		t.Fatalf("AddPath should not error with SkipBrokenSymlinks: %v", err)
	}

	// The real file should be present.
	if _, ok := dn.files["real.txt"]; !ok {
		t.Error("real.txt should be present")
	}

	// The broken symlink should be skipped.
	if _, ok := dn.files["broken.txt"]; ok {
		t.Error("broken.txt should have been skipped")
	}
}

func TestAddPath_ExcludePatterns_Good(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "app.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "debug.log"), []byte("log data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "error.log"), []byte("error data"), 0644); err != nil {
		t.Fatal(err)
	}

	dn := New()
	err := dn.AddPath(dir, AddPathOptions{
		ExcludePatterns: []string{"*.log"},
	})
	if err != nil {
		t.Fatalf("AddPath failed: %v", err)
	}

	// app.go should be present.
	if _, ok := dn.files["app.go"]; !ok {
		t.Error("app.go should be present")
	}

	// .log files should be excluded.
	if _, ok := dn.files["debug.log"]; ok {
		t.Error("debug.log should have been excluded")
	}
	if _, ok := dn.files["error.log"]; ok {
		t.Error("error.log should have been excluded")
	}
}

func TestAddPath_Bad(t *testing.T) {
	dn := New()
	err := dn.AddPath("/nonexistent/path/that/does/not/exist", AddPathOptions{})
	if err == nil {
		t.Fatal("expected error for nonexistent directory, got nil")
	}
}

func TestAddPath_ValidSymlink_Good(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks not reliably supported on Windows")
	}

	dir := t.TempDir()

	// Create a real file.
	if err := os.WriteFile(filepath.Join(dir, "target.txt"), []byte("target content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid symlink pointing to the real file.
	if err := os.Symlink("target.txt", filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}

	// Default behavior (FollowSymlinks=false): store as symlink.
	dn := New()
	err := dn.AddPath(dir, AddPathOptions{})
	if err != nil {
		t.Fatalf("AddPath failed: %v", err)
	}

	// The target file should be a regular file.
	targetFile, ok := dn.files["target.txt"]
	if !ok {
		t.Fatal("target.txt not found")
	}
	if targetFile.isSymlink() {
		t.Error("target.txt should not be a symlink")
	}
	if string(targetFile.content) != "target content" {
		t.Errorf("expected content 'target content', got %q", targetFile.content)
	}

	// The symlink should be stored as a symlink entry.
	linkFile, ok := dn.files["link.txt"]
	if !ok {
		t.Fatal("link.txt not found")
	}
	if !linkFile.isSymlink() {
		t.Error("link.txt should be a symlink")
	}
	if linkFile.symlink != "target.txt" {
		t.Errorf("expected symlink target 'target.txt', got %q", linkFile.symlink)
	}

	// Test with FollowSymlinks=true: store as regular file with target content.
	dn2 := New()
	err = dn2.AddPath(dir, AddPathOptions{FollowSymlinks: true})
	if err != nil {
		t.Fatalf("AddPath with FollowSymlinks failed: %v", err)
	}

	linkFile2, ok := dn2.files["link.txt"]
	if !ok {
		t.Fatal("link.txt not found with FollowSymlinks")
	}
	if linkFile2.isSymlink() {
		t.Error("link.txt should NOT be a symlink when FollowSymlinks is true")
	}
	if string(linkFile2.content) != "target content" {
		t.Errorf("expected content 'target content', got %q", linkFile2.content)
	}
}

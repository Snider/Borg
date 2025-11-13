package datanode

import (
	"io/fs"
	"os"
	"reflect"
	"sort"
	"testing"
)

func TestDataNode(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))
	dn.AddData("bar/baz.txt", []byte("baz"))
	dn.AddData("bar/qux.txt", []byte("qux"))

	// Test Open
	file, err := dn.Open("foo.txt")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	file.Close()

	_, err = dn.Open("nonexistent.txt")
	if err == nil {
		t.Fatalf("Expected error opening nonexistent file, got nil")
	}

	// Test Stat
	info, err := dn.Stat("bar/baz.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Name() != "baz.txt" {
		t.Errorf("Expected name baz.txt, got %s", info.Name())
	}
	if info.Size() != 3 {
		t.Errorf("Expected size 3, got %d", info.Size())
	}
	if info.IsDir() {
		t.Errorf("Expected baz.txt to not be a directory")
	}

	dirInfo, err := dn.Stat("bar")
	if err != nil {
		t.Fatalf("Stat directory failed: %v", err)
	}
	if !dirInfo.IsDir() {
		t.Errorf("Expected 'bar' to be a directory")
	}

	// Test Exists
	exists, err := dn.Exists("foo.txt")
	if err != nil || !exists {
		t.Errorf("Expected foo.txt to exist, err: %v", err)
	}
	exists, err = dn.Exists("bar")
	if err != nil || !exists {
		t.Errorf("Expected 'bar' directory to exist, err: %v", err)
	}
	exists, err = dn.Exists("nonexistent")
	if err != nil || exists {
		t.Errorf("Expected 'nonexistent' to not exist, err: %v", err)
	}

	// Test ReadDir
	entries, err := dn.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	expectedRootEntries := []string{"bar", "foo.txt"}
	if len(entries) != len(expectedRootEntries) {
		t.Errorf("Expected %d entries in root, got %d", len(expectedRootEntries), len(entries))
	}
	var rootEntryNames []string
	for _, e := range entries {
		rootEntryNames = append(rootEntryNames, e.Name())
	}
	sort.Strings(rootEntryNames)
	if !reflect.DeepEqual(rootEntryNames, expectedRootEntries) {
		t.Errorf("Expected entries %v, got %v", expectedRootEntries, rootEntryNames)
	}

	barEntries, err := dn.ReadDir("bar")
	if err != nil {
		t.Fatalf("ReadDir('bar') failed: %v", err)
	}
	expectedBarEntries := []string{"baz.txt", "qux.txt"}
	if len(barEntries) != len(expectedBarEntries) {
		t.Errorf("Expected %d entries in 'bar', got %d", len(expectedBarEntries), len(barEntries))
	}

	// Test Walk
	var paths []string
	dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
		paths = append(paths, path)
		return nil
	})
	expectedPaths := []string{".", "bar", "bar/baz.txt", "bar/qux.txt", "foo.txt"}
	sort.Strings(paths)
	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Walk expected paths %v, got %v", expectedPaths, paths)
	}

	// Test CopyFile
	tmpfile, err := os.CreateTemp("", "datanode-test-")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	err = dn.CopyFile("foo.txt", tmpfile.Name(), 0644)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "foo" {
		t.Errorf("Expected foo, got %s", string(content))
	}
}

package datanode

import (
	"io/fs"
	"reflect"
	"sort"
	"testing"
)

func TestDataNodeFS(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))
	dn.AddData("bar/baz.txt", []byte("baz"))
	dn.AddData("bar/qux.txt", []byte("qux"))

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

	_, err = dn.Stat("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
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
}

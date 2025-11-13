package datanode

import (
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

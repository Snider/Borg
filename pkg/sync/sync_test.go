package sync

import (
	"io/fs"
	"reflect"
	"sort"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestSync_Append(t *testing.T) {
	a := datanode.New()
	a.AddData("file1.txt", []byte("hello"))
	a.AddData("file2.txt", []byte("world"))

	b := datanode.New()
	b.AddData("file1.txt", []byte("different"))
	b.AddData("file3.txt", []byte("new"))

	result, err := Sync(a, b, AppendStrategy)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	expectedFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	assertDataNodeFiles(t, result, expectedFiles)
}

func TestSync_Mirror(t *testing.T) {
	a := datanode.New()
	a.AddData("file1.txt", []byte("hello"))
	a.AddData("file2.txt", []byte("world"))

	b := datanode.New()
	b.AddData("file3.txt", []byte("new"))

	result, err := Sync(a, b, MirrorStrategy)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	expectedFiles := []string{"file3.txt"}
	assertDataNodeFiles(t, result, expectedFiles)
}

func TestSync_Update(t *testing.T) {
	a := datanode.New()
	a.AddData("file1.txt", []byte("hello"))
	a.AddData("file2.txt", []byte("world"))

	b := datanode.New()
	b.AddData("file1.txt", []byte("updated"))
	b.AddData("file3.txt", []byte("new"))

	result, err := Sync(a, b, UpdateStrategy)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	expectedFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	assertDataNodeFiles(t, result, expectedFiles)
}

func assertDataNodeFiles(t *testing.T, dn *datanode.DataNode, expected []string) {
	t.Helper()
	var actual []string
	dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			actual = append(actual, path)
		}
		return nil
	})
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected files %v, got %v", expected, actual)
	}
}

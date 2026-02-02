package diff

import (
	"reflect"
	"sort"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestCompare_Good(t *testing.T) {
	a := datanode.New()
	a.AddData("file1.txt", []byte("hello"))
	a.AddData("file2.txt", []byte("world"))

	b := datanode.New()
	b.AddData("file1.txt", []byte("hello"))
	b.AddData("file2.txt", []byte("world"))

	diff, err := Compare(a, b)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if len(diff.Added) != 0 || len(diff.Removed) != 0 || len(diff.Modified) != 0 {
		t.Errorf("Expected no differences, but got %+v", diff)
	}
}

func TestCompare_Bad(t *testing.T) {
	a := datanode.New()
	a.AddData("file1.txt", []byte("hello"))
	a.AddData("file2.txt", []byte("world"))
	a.AddData("file3.txt", []byte("old"))

	b := datanode.New()
	b.AddData("file1.txt", []byte("hello"))
	b.AddData("file3.txt", []byte("new"))
	b.AddData("file4.txt", []byte("added"))

	diff, err := Compare(a, b)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	sort.Strings(diff.Modified)

	expectedAdded := []string{"file4.txt"}
	expectedRemoved := []string{"file2.txt"}
	expectedModified := []string{"file3.txt"}

	if !reflect.DeepEqual(diff.Added, expectedAdded) {
		t.Errorf("Expected Added %v, got %v", expectedAdded, diff.Added)
	}
	if !reflect.DeepEqual(diff.Removed, expectedRemoved) {
		t.Errorf("Expected Removed %v, got %v", expectedRemoved, diff.Removed)
	}
	if !reflect.DeepEqual(diff.Modified, expectedModified) {
		t.Errorf("Expected Modified %v, got %v", expectedModified, diff.Modified)
	}
}

func TestCompare_Ugly(t *testing.T) {
	a := datanode.New()
	b := datanode.New()

	diff, err := Compare(a, b)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if len(diff.Added) != 0 || len(diff.Removed) != 0 || len(diff.Modified) != 0 {
		t.Errorf("Expected no differences for empty datanodes, but got %+v", diff)
	}
}

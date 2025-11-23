package datanode

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestNew_Good(t *testing.T) {
	dn := New()
	if dn == nil {
		t.Fatal("New() returned nil")
	}
	if dn.files == nil {
		t.Error("New() did not initialize the files map")
	}
}

func TestAddData_Good(t *testing.T) {
	dn := New()
	path := "foo.txt"
	data := []byte("foo")
	dn.AddData(path, data)

	file, ok := dn.files[path]
	if !ok {
		t.Fatalf("file %q not found in datanode", path)
	}
	if string(file.content) != string(data) {
		t.Errorf("expected data %q, got %q", data, file.content)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatalf("file.Stat() failed: %v", err)
	}
	if info.Name() != "foo.txt" {
		t.Errorf("expected name foo.txt, got %s", info.Name())
	}
}

func TestAddData_Ugly(t *testing.T) {
	t.Run("Overwrite", func(t *testing.T) {
		dn := New()
		dn.AddData("foo.txt", []byte("foo"))
		dn.AddData("foo.txt", []byte("bar"))

		file, _ := dn.files["foo.txt"]
		if string(file.content) != "bar" {
			t.Errorf("expected data to be overwritten to 'bar', got %q", file.content)
		}
	})

	t.Run("Weird Path", func(t *testing.T) {
		dn := New()
		// path.Clean treats "a/../b/./c.txt" as "b/c.txt" but our implementation is simpler
		// and doesn't handle `..`. Let's test what it does handle.
		path := "./b/./c.txt"
		dn.AddData(path, []byte("c"))
		if _, ok := dn.files["./b/./c.txt"]; !ok {
			t.Errorf("expected path to be stored as is")
		}
	})
}

func TestOpen_Good(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))
	file, err := dn.Open("foo.txt")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer file.Close()
}

func TestOpen_Bad(t *testing.T) {
	dn := New()
	_, err := dn.Open("nonexistent.txt")
	if err == nil {
		t.Fatal("expected error opening nonexistent file, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected fs.ErrNotExist, got %v", err)
	}
}

func TestOpen_Ugly(t *testing.T) {
	dn := New()
	dn.AddData("bar/baz.txt", []byte("baz"))
	file, err := dn.Open("bar") // Opening a directory
	if err != nil {
		t.Fatalf("expected no error when opening a directory, got %v", err)
	}
	defer file.Close()

	// Reading from a directory should fail
	_, err = file.Read(make([]byte, 1))
	if err == nil {
		t.Fatal("expected error reading from a directory, got nil")
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || pathErr.Err != fs.ErrInvalid {
		t.Errorf("expected fs.ErrInvalid when reading a directory, got %v", err)
	}
}

func TestStat_Good(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))
	dn.AddData("bar/baz.txt", []byte("baz"))

	// Test file
	info, err := dn.Stat("bar/baz.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Name() != "baz.txt" {
		t.Errorf("expected name baz.txt, got %s", info.Name())
	}
	if info.Size() != 3 {
		t.Errorf("expected size 3, got %d", info.Size())
	}
	if info.IsDir() {
		t.Error("expected baz.txt to not be a directory")
	}

	// Test directory
	dirInfo, err := dn.Stat("bar")
	if err != nil {
		t.Fatalf("Stat directory failed: %v", err)
	}
	if !dirInfo.IsDir() {
		t.Error("expected 'bar' to be a directory")
	}
	if dirInfo.Name() != "bar" {
		t.Errorf("expected dir name 'bar', got %s", dirInfo.Name())
	}
}

func TestStat_Bad(t *testing.T) {
	dn := New()
	_, err := dn.Stat("nonexistent")
	if err == nil {
		t.Fatal("expected error stating nonexistent file, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected fs.ErrNotExist, got %v", err)
	}
}

func TestStat_Ugly(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))

	// Test root
	info, err := dn.Stat(".")
	if err != nil {
		t.Fatalf("Stat('.') failed: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected '.' to be a directory")
	}
	if info.Name() != "." {
		t.Errorf("expected name '.', got %s", info.Name())
	}
}

func TestExists_Good(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))
	dn.AddData("bar/baz.txt", []byte("baz"))

	exists, err := dn.Exists("foo.txt")
	if err != nil || !exists {
		t.Errorf("expected foo.txt to exist, err: %v", err)
	}

	exists, err = dn.Exists("bar")
	if err != nil || !exists {
		t.Errorf("expected 'bar' directory to exist, err: %v", err)
	}
}

func TestExists_Bad(t *testing.T) {
	dn := New()
	exists, err := dn.Exists("nonexistent")
	if err != nil {
		t.Errorf("unexpected error for nonexistent file: %v", err)
	}
	if exists {
		t.Error("expected 'nonexistent' to not exist")
	}
}

func TestExists_Ugly(t *testing.T) {
	dn := New()
	dn.AddData("dummy.txt", []byte("dummy"))
	// Test root
	exists, err := dn.Exists(".")
	if err != nil || !exists {
		t.Error("expected root '.' to exist")
	}
	// Test empty path
	exists, err = dn.Exists("")
	if err != nil {
		// our stat treats "" as "."
		if !strings.Contains(err.Error(), "exists") {
			t.Errorf("unexpected error for empty path: %v", err)
		}
	}
	if !exists {
		t.Error("expected empty path '' to exist (as root)")
	}
}

func TestReadDir_Good(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))
	dn.AddData("bar/baz.txt", []byte("baz"))
	dn.AddData("bar/qux.txt", []byte("qux"))

	// Read root
	entries, err := dn.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	expectedRootEntries := []string{"bar", "foo.txt"}
	entryNames := toSortedNames(entries)
	if !reflect.DeepEqual(entryNames, expectedRootEntries) {
		t.Errorf("expected entries %v, got %v", expectedRootEntries, entryNames)
	}

	// Read subdirectory
	barEntries, err := dn.ReadDir("bar")
	if err != nil {
		t.Fatalf("ReadDir('bar') failed: %v", err)
	}
	expectedBarEntries := []string{"baz.txt", "qux.txt"}
	barEntryNames := toSortedNames(barEntries)
	if !reflect.DeepEqual(barEntryNames, expectedBarEntries) {
		t.Errorf("expected entries %v, got %v", expectedBarEntries, barEntryNames)
	}
}

func TestReadDir_Bad(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))

	// Read nonexistent dir
	entries, err := dn.ReadDir("nonexistent")
	if err != nil {
		t.Fatalf("expected no error reading nonexistent dir, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for nonexistent dir, got %d", len(entries))
	}

	// Read file
	_, err = dn.ReadDir("foo.txt")
	if err == nil {
		t.Fatal("expected error reading a file")
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || pathErr.Err != fs.ErrInvalid {
		t.Errorf("expected fs.ErrInvalid when reading a file, got %v", err)
	}
}

func TestReadDir_Ugly(t *testing.T) {
	dn := New()
	dn.AddData("bar/baz.txt", []byte("baz"))
	dn.AddData("empty_dir/", nil)

	// Read dir with another dir but no files
	entries, err := dn.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	expected := []string{"bar"} // empty_dir/ is ignored by AddData
	names := toSortedNames(entries)
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected %v, got %v", expected, names)
	}
}

func TestWalk_Good(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))
	dn.AddData("bar/baz.txt", []byte("baz"))
	dn.AddData("bar/qux.txt", []byte("qux"))

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

func TestWalk_Bad(t *testing.T) {
	dn := New()
	// Walk non-existent path. fs.WalkDir will call the func with the error.
	var called bool
	err := dn.Walk("nonexistent", func(path string, d fs.DirEntry, err error) error {
		called = true
		if err == nil {
			t.Error("expected error for nonexistent path")
		}
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("unexpected error message: %v", err)
		}
		return err // propagate error
	})
	if !called {
		t.Fatal("walk function was not called for nonexistent root")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected Walk to return fs.ErrNotExist, got %v", err)
	}
}

func TestWalk_Ugly(t *testing.T) {
	dn := New()
	dn.AddData("a/b.txt", []byte("b"))
	dn.AddData("a/c.txt", []byte("c"))

	// Test stopping walk
	walkErr := errors.New("stop walking")
	var paths []string
	err := dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if path == "a/b.txt" {
			return walkErr
		}
		paths = append(paths, path)
		return nil
	})

	if err != walkErr {
		t.Errorf("expected walk to return the callback error, got %v", err)
	}
}

func TestWalk_Options(t *testing.T) {
	dn := New()
	dn.AddData("root.txt", []byte("root"))
	dn.AddData("a/a1.txt", []byte("a1"))
	dn.AddData("a/b/b1.txt", []byte("b1"))
	dn.AddData("c/c1.txt", []byte("c1"))

	t.Run("MaxDepth", func(t *testing.T) {
		var paths []string
		err := dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
			paths = append(paths, path)
			return nil
		}, WalkOptions{MaxDepth: 1})
		if err != nil {
			t.Fatalf("Walk failed: %v", err)
		}
		expected := []string{".", "a", "c", "root.txt"}
		sort.Strings(paths)
		if !reflect.DeepEqual(paths, expected) {
			t.Errorf("expected paths %v, got %v", expected, paths)
		}
	})

	t.Run("Filter", func(t *testing.T) {
		var paths []string
		err := dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
			paths = append(paths, path)
			return nil
		}, WalkOptions{Filter: func(path string, d fs.DirEntry) bool {
			return !strings.HasPrefix(path, "a")
		}})
		if err != nil {
			t.Fatalf("Walk failed: %v", err)
		}
		expected := []string{".", "c", "c/c1.txt", "root.txt"}
		sort.Strings(paths)
		if !reflect.DeepEqual(paths, expected) {
			t.Errorf("expected paths %v, got %v", expected, paths)
		}
	})

	t.Run("SkipErrors", func(t *testing.T) {
		// Mock a walk failure by passing a non-existent root with SkipErrors.
		// Normally, WalkDir calls fn with an error for the root if it doesn't exist.
		var called bool
		err := dn.Walk("nonexistent", func(path string, d fs.DirEntry, err error) error {
			called = true
			return err
		}, WalkOptions{SkipErrors: true})

		if err != nil {
			t.Errorf("expected no error with SkipErrors, got %v", err)
		}
		if called {
			t.Error("callback should NOT be called if error is skipped internally")
		}
	})
}

func TestCopyFile_Good(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))

	tmpfile := filepath.Join(t.TempDir(), "test.txt")
	err := dn.CopyFile("foo.txt", tmpfile, 0644)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	content, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "foo" {
		t.Errorf("expected foo, got %s", string(content))
	}
}

func TestCopyFile_Bad(t *testing.T) {
	dn := New()
	tmpfile := filepath.Join(t.TempDir(), "test.txt")

	// Source does not exist
	err := dn.CopyFile("nonexistent.txt", tmpfile, 0644)
	if err == nil {
		t.Fatal("expected error for nonexistent source file")
	}

	// Destination is not writable
	dn.AddData("foo.txt", []byte("foo"))
	err = dn.CopyFile("foo.txt", "/nonexistent_dir/test.txt", 0644)
	if err == nil {
		t.Fatal("expected error for unwritable destination")
	}
}

func TestCopyFile_Ugly(t *testing.T) {
	dn := New()
	dn.AddData("bar/baz.txt", []byte("baz"))
	tmpfile := filepath.Join(t.TempDir(), "test.txt")

	// Attempting to copy a directory
	err := dn.CopyFile("bar", tmpfile, 0644)
	if err == nil {
		t.Fatal("expected error when trying to copy a directory")
	}
}

func TestToTar_Good(t *testing.T) {
	dn := New()
	dn.AddData("foo.txt", []byte("foo"))
	dn.AddData("bar/baz.txt", []byte("baz"))

	tarball, err := dn.ToTar()
	if err != nil {
		t.Fatalf("ToTar failed: %v", err)
	}
	if len(tarball) == 0 {
		t.Fatal("expected non-empty tarball")
	}

	// Verify tar content
	tr := tar.NewReader(bytes.NewReader(tarball))
	files := make(map[string]string)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar.Next failed: %v", err)
		}
		content, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read tar content failed: %v", err)
		}
		files[header.Name] = string(content)
	}

	if files["foo.txt"] != "foo" {
		t.Errorf("expected foo.txt content 'foo', got %q", files["foo.txt"])
	}
	if files["bar/baz.txt"] != "baz" {
		t.Errorf("expected bar/baz.txt content 'baz', got %q", files["bar/baz.txt"])
	}
}

func TestFromTar_Good(t *testing.T) {
	// Create a tarball
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	files := []struct{ Name, Body string }{
		{"foo.txt", "foo"},
		{"bar/baz.txt", "baz"},
	}
	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Body)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("WriteHeader failed: %v", err)
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	dn, err := FromTar(buf.Bytes())
	if err != nil {
		t.Fatalf("FromTar failed: %v", err)
	}

	// Verify DataNode content
	exists, _ := dn.Exists("foo.txt")
	if !exists {
		t.Error("foo.txt missing")
	}
	exists, _ = dn.Exists("bar/baz.txt")
	if !exists {
		t.Error("bar/baz.txt missing")
	}
}

func TestTarRoundTrip_Good(t *testing.T) {
	dn1 := New()
	dn1.AddData("a.txt", []byte("a"))
	dn1.AddData("b/c.txt", []byte("c"))

	tarball, err := dn1.ToTar()
	if err != nil {
		t.Fatalf("ToTar failed: %v", err)
	}

	dn2, err := FromTar(tarball)
	if err != nil {
		t.Fatalf("FromTar failed: %v", err)
	}

	// Verify dn2 matches dn1
	exists, _ := dn2.Exists("a.txt")
	if !exists {
		t.Error("a.txt missing in dn2")
	}
	exists, _ = dn2.Exists("b/c.txt")
	if !exists {
		t.Error("b/c.txt missing in dn2")
	}
}

func TestFromTar_Bad(t *testing.T) {
	// Pass invalid data (truncated header)
	// A valid tar header is 512 bytes.
	truncated := make([]byte, 100)
	_, err := FromTar(truncated)
	if err == nil {
		t.Error("expected error for truncated tar header, got nil")
	} else if err != io.EOF && err != io.ErrUnexpectedEOF {
		// Verify it's some sort of read error or EOF related
		// Depending on implementation details of archive/tar
	}
}

func toSortedNames(entries []fs.DirEntry) []string {
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

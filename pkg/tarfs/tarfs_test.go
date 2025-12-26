package tarfs

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"testing"
	"time"
)

// createTestTar creates a tar archive with the given files in rootfs/ prefix
func createTestTar(files map[string][]byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for name, content := range files {
		hdr := &tar.Header{
			Name:    "rootfs/" + name,
			Mode:    0644,
			Size:    int64(len(content)),
			ModTime: time.Now(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(content); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func TestNew(t *testing.T) {
	t.Run("valid tar", func(t *testing.T) {
		files := map[string][]byte{
			"test.txt":        []byte("Hello, World!"),
			"subdir/file.txt": []byte("Nested file"),
		}
		tarData, err := createTestTar(files)
		if err != nil {
			t.Fatalf("Failed to create test tar: %v", err)
		}

		fs, err := New(tarData)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if fs == nil {
			t.Fatal("New() returned nil")
		}

		if len(fs.files) != 2 {
			t.Errorf("Expected 2 files, got %d", len(fs.files))
		}
	})

	t.Run("empty tar", func(t *testing.T) {
		buf := new(bytes.Buffer)
		tw := tar.NewWriter(buf)
		tw.Close()

		fs, err := New(buf.Bytes())
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if len(fs.files) != 0 {
			t.Errorf("Expected 0 files, got %d", len(fs.files))
		}
	})

	t.Run("invalid tar", func(t *testing.T) {
		_, err := New([]byte("not a tar file"))
		if err == nil {
			t.Error("Expected error for invalid tar data")
		}
	})

	t.Run("files without rootfs prefix are ignored", func(t *testing.T) {
		buf := new(bytes.Buffer)
		tw := tar.NewWriter(buf)

		// Add file without rootfs/ prefix
		hdr := &tar.Header{
			Name: "outside.txt",
			Mode: 0644,
			Size: 5,
		}
		tw.WriteHeader(hdr)
		tw.Write([]byte("hello"))

		// Add file with rootfs/ prefix
		hdr = &tar.Header{
			Name: "rootfs/inside.txt",
			Mode: 0644,
			Size: 5,
		}
		tw.WriteHeader(hdr)
		tw.Write([]byte("world"))

		tw.Close()

		fs, err := New(buf.Bytes())
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		// Only the rootfs file should be included
		if len(fs.files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(fs.files))
		}

		if _, ok := fs.files["inside.txt"]; !ok {
			t.Error("Expected 'inside.txt' to be in files")
		}
	})
}

func TestTarFS_Open(t *testing.T) {
	files := map[string][]byte{
		"test.txt":        []byte("Hello, World!"),
		"subdir/file.txt": []byte("Nested file"),
	}
	tarData, err := createTestTar(files)
	if err != nil {
		t.Fatalf("Failed to create test tar: %v", err)
	}

	fs, err := New(tarData)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Run("existing file", func(t *testing.T) {
		file, err := fs.Open("test.txt")
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if string(content) != "Hello, World!" {
			t.Errorf("Got %q, want %q", string(content), "Hello, World!")
		}
	})

	t.Run("existing file with leading slash", func(t *testing.T) {
		file, err := fs.Open("/test.txt")
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if string(content) != "Hello, World!" {
			t.Errorf("Got %q, want %q", string(content), "Hello, World!")
		}
	})

	t.Run("nested file", func(t *testing.T) {
		file, err := fs.Open("subdir/file.txt")
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if string(content) != "Nested file" {
			t.Errorf("Got %q, want %q", string(content), "Nested file")
		}
	})

	t.Run("non-existing file", func(t *testing.T) {
		_, err := fs.Open("nonexistent.txt")
		if err != os.ErrNotExist {
			t.Errorf("Expected os.ErrNotExist, got %v", err)
		}
	})

	t.Run("multiple reads reset position", func(t *testing.T) {
		// First read
		file1, _ := fs.Open("test.txt")
		content1, _ := io.ReadAll(file1)
		file1.Close()

		// Second read should work too
		file2, _ := fs.Open("test.txt")
		content2, _ := io.ReadAll(file2)
		file2.Close()

		if string(content1) != string(content2) {
			t.Errorf("Multiple reads returned different content")
		}
	})
}

func TestTarFile_Methods(t *testing.T) {
	files := map[string][]byte{
		"test.txt": []byte("Hello, World!"),
	}
	tarData, err := createTestTar(files)
	if err != nil {
		t.Fatalf("Failed to create test tar: %v", err)
	}

	fs, err := New(tarData)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	file, err := fs.Open("test.txt")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer file.Close()

	t.Run("Read", func(t *testing.T) {
		buf := make([]byte, 5)
		n, err := file.Read(buf)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if n != 5 {
			t.Errorf("Read() returned %d bytes, want 5", n)
		}
		if string(buf) != "Hello" {
			t.Errorf("Got %q, want %q", string(buf), "Hello")
		}
	})

	t.Run("Seek", func(t *testing.T) {
		pos, err := file.Seek(0, io.SeekStart)
		if err != nil {
			t.Fatalf("Seek() error = %v", err)
		}
		if pos != 0 {
			t.Errorf("Seek() returned position %d, want 0", pos)
		}

		pos, err = file.Seek(7, io.SeekStart)
		if err != nil {
			t.Fatalf("Seek() error = %v", err)
		}
		if pos != 7 {
			t.Errorf("Seek() returned position %d, want 7", pos)
		}

		buf := make([]byte, 6)
		file.Read(buf)
		if string(buf) != "World!" {
			t.Errorf("After seek, got %q, want %q", string(buf), "World!")
		}
	})

	t.Run("Readdir", func(t *testing.T) {
		_, err := file.Readdir(0)
		if err != os.ErrInvalid {
			t.Errorf("Readdir() should return os.ErrInvalid, got %v", err)
		}
	})

	t.Run("Stat", func(t *testing.T) {
		info, err := file.Stat()
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}

		if info.Name() != "test.txt" {
			t.Errorf("Name() = %q, want %q", info.Name(), "test.txt")
		}

		if info.Size() != 13 {
			t.Errorf("Size() = %d, want 13", info.Size())
		}

		if info.Mode() != 0444 {
			t.Errorf("Mode() = %v, want 0444", info.Mode())
		}

		if info.IsDir() {
			t.Error("IsDir() should be false")
		}

		if info.Sys() != nil {
			t.Error("Sys() should return nil")
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := file.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
}

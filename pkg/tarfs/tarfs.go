package tarfs

import (
	"archive/tar"
	"bytes"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// TarFS is a http.FileSystem that serves files from a tar archive.
type TarFS struct {
	files map[string]*tarFile
}

// New creates a new TarFS from a tar archive.
func New(data []byte) (*TarFS, error) {
	fs := &TarFS{
		files: make(map[string]*tarFile),
	}

	tr := tar.NewReader(bytes.NewReader(data))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(hdr.Name, "rootfs/") {
			content, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
			fs.files[strings.TrimPrefix(hdr.Name, "rootfs/")] = &tarFile{
				header:  hdr,
				content: bytes.NewReader(content),
				modTime: hdr.ModTime,
			}
		}
	}

	return fs, nil
}

// Open opens a file from the tar archive.
func (fs *TarFS) Open(name string) (http.File, error) {
	name = strings.TrimPrefix(name, "/")
	if file, ok := fs.files[name]; ok {
		// Reset the reader to the beginning of the file
		file.content.Seek(0, 0)
		return file, nil
	}

	return nil, os.ErrNotExist
}

// tarFile is a http.File that represents a file in a tar archive.
type tarFile struct {
	header  *tar.Header
	content *bytes.Reader
	modTime time.Time
}

// Close implements http.File Close with a no-op for tar-backed files.
func (f *tarFile) Close() error { return nil }

// Read implements io.Reader by delegating to the underlying bytes.Reader.
func (f *tarFile) Read(p []byte) (int, error) { return f.content.Read(p) }

// Seek implements io.Seeker by delegating to the underlying bytes.Reader.
func (f *tarFile) Seek(offset int64, whence int) (int64, error) {
	return f.content.Seek(offset, whence)
}

// Readdir is unsupported for files in the tar filesystem and returns os.ErrInvalid.
func (f *tarFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

// Stat returns a FileInfo describing the tar-backed file.
func (f *tarFile) Stat() (os.FileInfo, error) {
	return &tarFileInfo{
		name:    path.Base(f.header.Name),
		size:    f.header.Size,
		modTime: f.modTime,
	}, nil
}

// tarFileInfo is a os.FileInfo that represents a file in a tar archive.
type tarFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

// Name returns the base name of the tar file.
func (i *tarFileInfo) Name() string { return i.name }

// Size returns the size of the tar file in bytes.
func (i *tarFileInfo) Size() int64 { return i.size }

// Mode returns a read-only file mode for tar entries.
func (i *tarFileInfo) Mode() os.FileMode { return 0444 }

// ModTime returns the modification time recorded in the tar header.
func (i *tarFileInfo) ModTime() time.Time { return i.modTime }

// IsDir reports whether the entry is a directory (always false for files here).
func (i *tarFileInfo) IsDir() bool { return false }

// Sys returns underlying data source (unused for tar entries).
func (i *tarFileInfo) Sys() interface{} { return nil }

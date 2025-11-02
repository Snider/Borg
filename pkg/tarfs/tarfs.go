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
	files map[string]*tar.Header
	data  []byte
}

// New creates a new TarFS from a tar archive.
func New(data []byte) (*TarFS, error) {
	fs := &TarFS{
		files: make(map[string]*tar.Header),
		data:  data,
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
			fs.files[strings.TrimPrefix(hdr.Name, "rootfs/")] = hdr
		}
	}

	return fs, nil
}

// Open opens a file from the tar archive.
func (fs *TarFS) Open(name string) (http.File, error) {
	name = strings.TrimPrefix(name, "/")
	if hdr, ok := fs.files[name]; ok {
		// This is a bit inefficient, but it's the simplest way to
		// get the file content without pre-indexing everything.
		tr := tar.NewReader(bytes.NewReader(fs.data))
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			if h.Name == hdr.Name {
				return &tarFile{
					header:  hdr,
					content: tr,
					modTime: hdr.ModTime,
				}, nil
			}
		}
	}

	return nil, os.ErrNotExist
}

// tarFile is a http.File that represents a file in a tar archive.
type tarFile struct {
	header  *tar.Header
	content io.Reader
	modTime time.Time
}

func (f *tarFile) Close() error               { return nil }
func (f *tarFile) Read(p []byte) (int, error) { return f.content.Read(p) }
func (f *tarFile) Seek(offset int64, whence int) (int64, error) {
	return 0, io.ErrUnexpectedEOF
}

func (f *tarFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

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

func (i *tarFileInfo) Name() string       { return i.name }
func (i *tarFileInfo) Size() int64        { return i.size }
func (i *tarFileInfo) Mode() os.FileMode  { return 0444 }
func (i *tarFileInfo) ModTime() time.Time { return i.modTime }
func (i *tarFileInfo) IsDir() bool        { return false }
func (i *tarFileInfo) Sys() interface{}   { return nil }

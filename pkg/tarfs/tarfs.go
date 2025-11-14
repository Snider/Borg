// Package tarfs provides an http.FileSystem implementation that serves files
// from a tar archive. This is particularly useful for serving the contents of a
// .tim file's rootfs directly without unpacking it to disk.
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

// TarFS is an implementation of http.FileSystem that serves files from an
// in-memory tar archive. It specifically looks for files within a "rootfs/"
// directory in the archive, which is the convention used by .tim files.
type TarFS struct {
	files map[string]*tarFile
}

// New creates a new TarFS from a byte slice containing a tar archive. It
// parses the archive and stores the files from the "rootfs/" directory in
// memory.
//
// Example:
//
//	tarData, err := os.ReadFile("my-archive.tar")
//	if err != nil {
//		// handle error
//	}
//	fs, err := tarfs.New(tarData)
//	if err != nil {
//		// handle error
//	}
//	http.Handle("/", http.FileServer(fs))
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

// Open opens a file from the tar archive for reading. This is the implementation
// of the http.FileSystem interface.
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

func (f *tarFile) Close() error               { return nil }
func (f *tarFile) Read(p []byte) (int, error) { return f.content.Read(p) }
func (f *tarFile) Seek(offset int64, whence int) (int64, error) {
	return f.content.Seek(offset, whence)
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

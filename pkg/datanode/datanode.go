package datanode

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidPassword  = errors.New("invalid password")
	ErrPasswordRequired = errors.New("password required")
)

// DataNode is a filesystem that reads from a tar archive on demand.
type DataNode struct {
	archive io.ReaderAt
	files   map[string]*fileIndex
}

// fileIndex stores the metadata for a file in the archive.
type fileIndex struct {
	name    string
	offset  int64
	size    int64
	modTime time.Time
}

// New creates a new, empty DataNode.
func New(archive io.ReaderAt) *DataNode {
	return &DataNode{
		archive: archive,
		files:   make(map[string]*fileIndex),
	}
}

// FromTar creates a new DataNode from a tarball.
func FromTar(archive io.ReaderAt) (*DataNode, error) {
	dn := New(archive)
	if seeker, ok := archive.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	offset := int64(0)
	for {
		headerData := make([]byte, 512)
		_, err := archive.ReadAt(headerData, offset)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		header, err := tar.NewReader(bytes.NewReader(headerData)).Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		offset += 512
		if header.Typeflag == tar.TypeReg {
			dn.files[header.Name] = &fileIndex{
				name:    header.Name,
				offset:  offset,
				size:    header.Size,
				modTime: header.ModTime,
			}
			offset += header.Size
			if remainder := header.Size % 512; remainder != 0 {
				offset += 512 - remainder
			}
		}

	}

	return dn, nil
}

// ToTar serializes the DataNode to a tarball.
// This function will need to be re-implemented to read from the archive.
// For now, it will return an error.
func (d *DataNode) ToTar() ([]byte, error) {
	return nil, errors.New("ToTar is not implemented for streaming DataNodes")
}

// AddData is not supported for streaming DataNodes.
func (d *DataNode) AddData(name string, content []byte) {
	// This is a no-op for now.
}

// Open opens a file from the DataNode.
func (d *DataNode) Open(name string) (fs.File, error) {
	name = strings.TrimPrefix(name, "/")
	if file, ok := d.files[name]; ok {
		sectionReader := io.NewSectionReader(d.archive, file.offset, file.size)
		return &dataFileReader{
			file:   file,
			reader: sectionReader,
		}, nil
	}
	// Check if it's a directory
	prefix := name + "/"
	if name == "." || name == "" {
		prefix = ""
	}
	for p := range d.files {
		if strings.HasPrefix(p, prefix) {
			return &dirFile{path: name, modTime: time.Now()}, nil
		}
	}
	return nil, fs.ErrNotExist
}

// ReadDir reads and returns all directory entries for the named directory.
func (d *DataNode) ReadDir(name string) ([]fs.DirEntry, error) {
	name = strings.TrimPrefix(name, "/")
	if name == "." {
		name = ""
	}

	// Disallow reading a file as a directory.
	if info, err := d.Stat(name); err == nil && !info.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	entries := []fs.DirEntry{}
	seen := make(map[string]bool)

	prefix := ""
	if name != "" {
		prefix = name + "/"
	}

	for p := range d.files {
		if !strings.HasPrefix(p, prefix) {
			continue
		}

		relPath := strings.TrimPrefix(p, prefix)
		firstComponent := strings.Split(relPath, "/")[0]

		if seen[firstComponent] {
			continue
		}
		seen[firstComponent] = true

		if strings.Contains(relPath, "/") {
			// It's a directory
			dir := &dirInfo{name: firstComponent, modTime: time.Now()}
			entries = append(entries, fs.FileInfoToDirEntry(dir))
		} else {
			// It's a file
			file := d.files[p]
			info, _ := file.Stat()
			entries = append(entries, fs.FileInfoToDirEntry(info))
		}
	}

	// Sort for stable order in tests
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

// Stat returns the FileInfo structure describing file.
func (d *DataNode) Stat(name string) (fs.FileInfo, error) {
	name = strings.TrimPrefix(name, "/")
	if file, ok := d.files[name]; ok {
		return file.Stat()
	}
	// Check if it's a directory
	prefix := name + "/"
	if name == "." || name == "" {
		prefix = ""
	}
	for p := range d.files {
		if strings.HasPrefix(p, prefix) {
			return &dirInfo{name: path.Base(name), modTime: time.Now()}, nil
		}
	}

	return nil, fs.ErrNotExist
}

// ExistsOptions allows customizing the Exists check.
type ExistsOptions struct {
	WantType fs.FileMode
}

// Exists returns true if the file or directory exists.
func (d *DataNode) Exists(name string, opts ...ExistsOptions) (bool, error) {
	info, err := d.Stat(name)
	if err != nil {
		if err == fs.ErrNotExist || os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if len(opts) > 0 {
		if opts[0].WantType == fs.ModeDir && !info.IsDir() {
			return false, nil
		}
		if opts[0].WantType != fs.ModeDir && info.IsDir() {
			return false, nil
		}
	}
	return true, nil
}

// WalkOptions allows customizing the Walk behavior.
type WalkOptions struct {
	MaxDepth   int
	Filter     func(path string, d fs.DirEntry) bool
	SkipErrors bool
}

// Walk recursively descends the file tree rooted at root, calling fn for each file or directory.
func (d *DataNode) Walk(root string, fn fs.WalkDirFunc, opts ...WalkOptions) error {
	var maxDepth int
	var filter func(string, fs.DirEntry) bool
	var skipErrors bool
	if len(opts) > 0 {
		maxDepth = opts[0].MaxDepth
		filter = opts[0].Filter
_		skipErrors = opts[0].SkipErrors
	}

	return fs.WalkDir(d, root, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			if skipErrors {
				return nil
			}
			return fn(path, de, err)
		}
		if filter != nil && !filter(path, de) {
			if de.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// Process the entry first.
		if err := fn(path, de, nil); err != nil {
			return err
		}

		if maxDepth > 0 {
			// Calculate depth relative to root
			cleanedPath := strings.TrimPrefix(path, root)
			cleanedPath = strings.TrimPrefix(cleanedPath, "/")

			currentDepth := 0
			if path != root {
				if cleanedPath == "" {
					// This can happen if root is "bar" and path is "bar"
					currentDepth = 0
				} else {
					currentDepth = strings.Count(cleanedPath, "/") + 1
				}
			}

			if de.IsDir() && currentDepth >= maxDepth {
				return fs.SkipDir
			}
		}
		return nil
	})
}

// CopyFile copies a file from the DataNode to the local filesystem.
func (d *DataNode) CopyFile(sourcePath string, target string, perm os.FileMode) error {
	sourceFile, err := d.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, perm)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, sourceFile)
	return err
}

func (d *fileIndex) Stat() (fs.FileInfo, error) { return &dataFileInfo{file: d}, nil }

// dataFileInfo implements fs.FileInfo for a dataFile.
type dataFileInfo struct{ file *fileIndex }

func (d *dataFileInfo) Name() string       { return path.Base(d.file.name) }
func (d *dataFileInfo) Size() int64        { return d.file.size }
func (d *dataFileInfo) Mode() fs.FileMode  { return 0444 }
func (d *dataFileInfo) ModTime() time.Time { return d.file.modTime }
func (d *dataFileInfo) IsDir() bool        { return false }
func (d *dataFileInfo) Sys() interface{}   { return nil }

// dataFileReader implements fs.File for a dataFile.
type dataFileReader struct {
	file   *fileIndex
	reader io.ReaderAt
}

func (d *dataFileReader) Stat() (fs.FileInfo, error) { return d.file.Stat() }
func (d *dataFileReader) Read(p []byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.file.name, Err: fs.ErrInvalid}
}
func (d *dataFileReader) ReadAt(p []byte, off int64) (n int, err error) {
	return d.reader.ReadAt(p, off)
}
func (d *dataFileReader) Close() error { return nil }

// dirInfo implements fs.FileInfo for an implicit directory.
type dirInfo struct {
	name    string
	modTime time.Time
}

func (d *dirInfo) Name() string       { return d.name }
func (d *dirInfo) Size() int64        { return 0 }
func (d *dirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0555 }
func (d *dirInfo) ModTime() time.Time { return d.modTime }
func (d *dirInfo) IsDir() bool        { return true }
func (d *dirInfo) Sys() interface{}   { return nil }

// dirFile implements fs.File for a directory.
type dirFile struct {
	path    string
	modTime time.Time
}

func (d *dirFile) Stat() (fs.FileInfo, error) {
	return &dirInfo{name: path.Base(d.path), modTime: d.modTime}, nil
}
func (d *dirFile) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.path, Err: fs.ErrInvalid}
}
func (d *dirFile) Close() error { return nil }

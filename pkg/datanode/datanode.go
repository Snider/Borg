package datanode

import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

// DataNode is an in-memory filesystem that is compatible with fs.FS.
type DataNode struct {
	files map[string]*dataFile
}

// New creates a new, empty DataNode.
func New() *DataNode {
	return &DataNode{files: make(map[string]*dataFile)}
}

// FromTar creates a new DataNode from a tarball.
func FromTar(tarball []byte) (*DataNode, error) {
	dn := New()
	tarReader := tar.NewReader(bytes.NewReader(tarball))

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, err
			}
			dn.AddData(header.Name, data)
		}
	}

	return dn, nil
}

// ToTar serializes the DataNode to a tarball.
func (d *DataNode) ToTar() ([]byte, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for _, file := range d.files {
		hdr := &tar.Header{
			Name:    file.name,
			Mode:    0600,
			Size:    int64(len(file.content)),
			ModTime: file.modTime,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(file.content); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// AddData adds a file to the DataNode.
func (d *DataNode) AddData(name string, content []byte) {
	name = strings.TrimPrefix(name, "/")
	d.files[name] = &dataFile{
		name:    name,
		content: content,
		modTime: time.Now(),
	}
}

// Open opens a file from the DataNode.
func (d *DataNode) Open(name string) (fs.File, error) {
	name = strings.TrimPrefix(name, "/")
	if file, ok := d.files[name]; ok {
		return &dataFileReader{file: file}, nil
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
		skipErrors = opts[0].SkipErrors
	}

	return fs.WalkDir(d, root, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			if skipErrors {
				return nil
			}
			return fn(path, de, err)
		}
		if filter != nil && !filter(path, de) {
			return nil
		}
		if maxDepth > 0 {
			currentDepth := strings.Count(strings.TrimPrefix(path, root), "/")
			if de.IsDir() && currentDepth >= maxDepth {
				return fs.SkipDir
			}
		}
		return fn(path, de, nil)
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

// dataFile represents a file in the DataNode.
type dataFile struct {
	name    string
	content []byte
	modTime time.Time
}

// Stat returns a FileInfo describing the dataFile.
func (d *dataFile) Stat() (fs.FileInfo, error) { return &dataFileInfo{file: d}, nil }

// Read implements fs.File by returning EOF for write-only dataFile handles.
func (d *dataFile) Read(p []byte) (int, error) { return 0, io.EOF }

// Close is a no-op for in-memory dataFile values.
func (d *dataFile) Close() error { return nil }

// dataFileInfo implements fs.FileInfo for a dataFile.
type dataFileInfo struct{ file *dataFile }

// Name returns the base name of the data file.
func (d *dataFileInfo) Name() string { return path.Base(d.file.name) }

// Size returns the size of the data file in bytes.
func (d *dataFileInfo) Size() int64 { return int64(len(d.file.content)) }

// Mode returns the file mode bits for a read-only regular file.
func (d *dataFileInfo) Mode() fs.FileMode { return 0444 }

// ModTime returns the modification time of the data file.
func (d *dataFileInfo) ModTime() time.Time { return d.file.modTime }

// IsDir reports whether the FileInfo describes a directory (always false).
func (d *dataFileInfo) IsDir() bool { return false }

// Sys returns underlying data source (always nil).
func (d *dataFileInfo) Sys() interface{} { return nil }

// dataFileReader implements fs.File for a dataFile.
type dataFileReader struct {
	file   *dataFile
	reader *bytes.Reader
}

// Stat returns a FileInfo describing the underlying data file.
func (d *dataFileReader) Stat() (fs.FileInfo, error) { return d.file.Stat() }

// Read reads from the underlying byte slice, initializing the reader on first use.
func (d *dataFileReader) Read(p []byte) (int, error) {
	if d.reader == nil {
		d.reader = bytes.NewReader(d.file.content)
	}
	return d.reader.Read(p)
}

// Close is a no-op for in-memory readers.
func (d *dataFileReader) Close() error { return nil }

// dirInfo implements fs.FileInfo for an implicit directory.
type dirInfo struct {
	name    string
	modTime time.Time
}

// Name returns the directory name.
func (d *dirInfo) Name() string { return d.name }

// Size returns the size for a directory (always 0).
func (d *dirInfo) Size() int64 { return 0 }

// Mode returns the file mode bits indicating a read-only directory.
func (d *dirInfo) Mode() fs.FileMode { return fs.ModeDir | 0555 }

// ModTime returns the modification time of the directory.
func (d *dirInfo) ModTime() time.Time { return d.modTime }

// IsDir reports that this FileInfo describes a directory.
func (d *dirInfo) IsDir() bool { return true }

// Sys returns underlying data source (always nil).
func (d *dirInfo) Sys() interface{} { return nil }

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

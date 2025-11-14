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

// DataNode represents an in-memory filesystem, compatible with the standard
// library's io/fs.FS interface. It stores files and their contents in memory,
// making it useful for manipulating collections of files, such as those from
// a tar archive or a Git repository, without writing them to disk.
type DataNode struct {
	files map[string]*dataFile
}

// New creates and returns a new, empty DataNode. This is the starting point
// for building an in-memory filesystem.
//
// Example:
//
//	dn := datanode.New()
func New() *DataNode {
	return &DataNode{files: make(map[string]*dataFile)}
}

// FromTar creates a new DataNode by reading a tar archive. The tarball's
// contents are unpacked into the in-memory filesystem.
//
// Example:
//
//	tarData, err := os.ReadFile("my-archive.tar")
//	if err != nil {
//		// handle error
//	}
//	dn, err := datanode.FromTar(tarData)
//	if err != nil {
//		// handle error
//	}
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

// ToTar serializes the DataNode into a tar archive. This is useful for
// saving the in-memory filesystem to disk or for transmitting it over a
// network.
//
// Example:
//
//	tarData, err := dn.ToTar()
//	if err != nil {
//		// handle error
//	}
//	err = os.WriteFile("my-archive.tar", tarData, 0644)
//	if err != nil {
//		// handle error
//	}
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

// AddData adds a file to the DataNode with the given name and content. If the
// file already exists, it will be overwritten. Directory paths are created
// implicitly and do not need to be added separately.
//
// Example:
//
//	dn.AddData("my-file.txt", []byte("hello world"))
//	dn.AddData("my-dir/my-other-file.txt", []byte("hello again"))
func (d *DataNode) AddData(name string, content []byte) {
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		return
	}
	// Directories are implicit, so we don't store them.
	// A name ending in "/" is treated as a directory.
	if strings.HasSuffix(name, "/") {
		return
	}
	d.files[name] = &dataFile{
		name:    name,
		content: content,
		modTime: time.Now(),
	}
}

// Open opens a file from the DataNode for reading. It returns an fs.File,
// which can be used with standard library functions that operate on files.
// This method is part of the fs.FS interface implementation.
//
// Example:
//
//	file, err := dn.Open("my-file.txt")
//	if err != nil {
//		// handle error
//	}
//	defer file.Close()
//	content, err := io.ReadAll(file)
//	if err != nil {
//		// handle error
//	}
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

// ReadDir reads the named directory and returns a list of directory entries.
// This method is part of the fs.ReadDirFS interface implementation.
//
// Example:
//
//	entries, err := dn.ReadDir("my-dir")
//	if err != nil {
//		// handle error
//	}
//	for _, entry := range entries {
//		fmt.Println(entry.Name())
//	}
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

// Stat returns the fs.FileInfo structure describing the named file or directory.
// This method is part of the fs.StatFS interface implementation.
//
// Example:
//
//	info, err := dn.Stat("my-file.txt")
//	if err != nil {
//		// handle error
//	}
//	fmt.Println(info.Size())
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

// ExistsOptions provides options for customizing the behavior of the Exists
// method.
type ExistsOptions struct {
	// WantType specifies the desired file type (e.g., fs.ModeDir for a
	// directory). If the file exists but is not of the desired type, Exists
	// will return false.
	WantType fs.FileMode
}

// Exists checks if a file or directory at the given path exists in the DataNode.
// It can optionally check if the file is of a specific type (e.g., a directory).
//
// Example:
//
//	// Check if a file exists
//	exists, err := dn.Exists("my-file.txt")
//	if err != nil {
//		// handle error
//	}
//
//	// Check if a directory exists
//	exists, err = dn.Exists("my-dir", datanode.ExistsOptions{WantType: fs.ModeDir})
//	if err != nil {
//		// handle error
//	}
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

// WalkOptions provides options for customizing the behavior of the Walk method.
type WalkOptions struct {
	// MaxDepth limits the depth of the walk. A value of 0 means no limit.
	MaxDepth int
	// Filter is a function that can be used to skip files or directories. If
	// the function returns false for an entry, that entry is skipped. If the
	// entry is a directory, the entire subdirectory is skipped.
	Filter func(path string, d fs.DirEntry) bool
	// SkipErrors causes the walk to continue when an error is encountered.
	SkipErrors bool
}

// Walk walks the in-memory file tree rooted at root, calling fn for each file or
// directory in the tree, including root. The walk is depth-first.
//
// Example:
//
//	err := dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
//		if err != nil {
//			return err
//		}
//		fmt.Println(path)
//		return nil
//	})
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

// CopyFile copies a file from the DataNode to a specified path on the local
// filesystem.
//
// Example:
//
//	err := dn.CopyFile("my-file.txt", "/tmp/my-file.txt", 0644)
//	if err != nil {
//		// handle error
//	}
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

func (d *dataFile) Stat() (fs.FileInfo, error) { return &dataFileInfo{file: d}, nil }
func (d *dataFile) Read(p []byte) (int, error) { return 0, io.EOF }
func (d *dataFile) Close() error               { return nil }

// dataFileInfo implements fs.FileInfo for a dataFile.
type dataFileInfo struct{ file *dataFile }

func (d *dataFileInfo) Name() string       { return path.Base(d.file.name) }
func (d *dataFileInfo) Size() int64        { return int64(len(d.file.content)) }
func (d *dataFileInfo) Mode() fs.FileMode  { return 0444 }
func (d *dataFileInfo) ModTime() time.Time { return d.file.modTime }
func (d *dataFileInfo) IsDir() bool        { return false }
func (d *dataFileInfo) Sys() interface{}   { return nil }

// dataFileReader implements fs.File for a dataFile.
type dataFileReader struct {
	file   *dataFile
	reader *bytes.Reader
}

func (d *dataFileReader) Stat() (fs.FileInfo, error) { return d.file.Stat() }
func (d *dataFileReader) Read(p []byte) (int, error) {
	if d.reader == nil {
		d.reader = bytes.NewReader(d.file.content)
	}
	return d.reader.Read(p)
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

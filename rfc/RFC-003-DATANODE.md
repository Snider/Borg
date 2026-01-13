# RFC-003: DataNode In-Memory Filesystem

**Status**: Draft
**Author**: [Snider](https://github.com/Snider/)
**Created**: 2026-01-13
**License**: EUPL-1.2

---

## Abstract

DataNode is an in-memory filesystem abstraction implementing Go's `fs.FS` interface. It provides the foundation for collecting, manipulating, and serializing file trees without touching disk.

## 1. Overview

DataNode serves as the core data structure for:
- Collecting files from various sources (GitHub, websites, PWAs)
- Building container filesystems (TIM rootfs)
- Serializing to/from tar archives
- Encrypting as TRIX format

## 2. Implementation

### 2.1 Core Type

```go
type DataNode struct {
    files map[string]*dataFile
}

type dataFile struct {
    name    string
    content []byte
    modTime time.Time
}
```

**Key insight**: DataNode uses a **flat key-value map**, not a nested tree structure. Paths are stored as keys directly, and directories are implicit (derived from path prefixes).

### 2.2 fs.FS Implementation

DataNode implements these interfaces:

| Interface | Method | Description |
|-----------|--------|-------------|
| `fs.FS` | `Open(name string)` | Returns fs.File for path |
| `fs.StatFS` | `Stat(name string)` | Returns fs.FileInfo |
| `fs.ReadDirFS` | `ReadDir(name string)` | Lists directory contents |

### 2.3 Internal Helper Types

```go
// File metadata
type dataFileInfo struct {
    name    string
    size    int64
    modTime time.Time
}
func (fi *dataFileInfo) Mode() fs.FileMode { return 0444 }  // Read-only

// Directory metadata
type dirInfo struct {
    name string
}
func (di *dirInfo) Mode() fs.FileMode { return fs.ModeDir | 0555 }

// File reader (implements fs.File)
type dataFileReader struct {
    info   *dataFileInfo
    reader *bytes.Reader
}

// Directory reader (implements fs.File)
type dirFile struct {
    info    *dirInfo
    entries []fs.DirEntry
    offset  int
}
```

## 3. Operations

### 3.1 Construction

```go
// Create empty DataNode
node := datanode.New()

// Returns: &DataNode{files: make(map[string]*dataFile)}
```

### 3.2 Adding Files

```go
// Add file with content
node.AddData("path/to/file.txt", []byte("content"))

// Trailing slashes are ignored (treated as directory indicator)
node.AddData("path/to/dir/", []byte(""))  // Stored as "path/to/dir"
```

**Note**: Parent directories are NOT explicitly created. They are implicit based on path prefixes.

### 3.3 File Access

```go
// Open file (fs.FS interface)
f, err := node.Open("path/to/file.txt")
if err != nil {
    // fs.ErrNotExist if not found
}
defer f.Close()
content, _ := io.ReadAll(f)

// Stat file
info, err := node.Stat("path/to/file.txt")
// info.Name(), info.Size(), info.ModTime(), info.Mode()

// Read directory
entries, err := node.ReadDir("path/to")
for _, entry := range entries {
    // entry.Name(), entry.IsDir(), entry.Type()
}
```

### 3.4 Walking

```go
err := fs.WalkDir(node, ".", func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        return err
    }
    if !d.IsDir() {
        // Process file
    }
    return nil
})
```

## 4. Path Semantics

### 4.1 Path Handling

- **Leading slashes stripped**: `/path/file` → `path/file`
- **Trailing slashes ignored**: `path/dir/` → `path/dir`
- **Forward slashes only**: Uses `/` regardless of OS
- **Case-sensitive**: `File.txt` ≠ `file.txt`
- **Direct lookup**: Paths stored as flat keys

### 4.2 Valid Paths

```
file.txt              → stored as "file.txt"
dir/file.txt          → stored as "dir/file.txt"
/absolute/path        → stored as "absolute/path" (leading / stripped)
path/to/dir/          → stored as "path/to/dir" (trailing / stripped)
```

### 4.3 Directory Detection

Directories are **implicit**. A directory exists if:
1. Any file path has it as a prefix
2. Example: Adding `a/b/c.txt` implicitly creates directories `a` and `a/b`

```go
// ReadDir finds directories by scanning all paths
func (dn *DataNode) ReadDir(name string) ([]fs.DirEntry, error) {
    // Scans all keys for matching prefix
    // Returns unique immediate children
}
```

## 5. Tar Serialization

### 5.1 ToTar

```go
tarBytes, err := node.ToTar()
```

**Format**:
- All files written as `tar.TypeReg` (regular files)
- Header Mode: **0600** (fixed, not original mode)
- No explicit directory entries
- ModTime preserved from dataFile

```go
// Serialization logic
for path, file := range dn.files {
    header := &tar.Header{
        Name:    path,
        Mode:    0600,          // Fixed mode
        Size:    int64(len(file.content)),
        ModTime: file.modTime,
        Typeflag: tar.TypeReg,
    }
    tw.WriteHeader(header)
    tw.Write(file.content)
}
```

### 5.2 FromTar

```go
node, err := datanode.FromTar(tarBytes)
```

**Parsing**:
- Only reads `tar.TypeReg` entries
- Ignores directory entries (`tar.TypeDir`)
- Stores path and content in flat map

```go
// Deserialization logic
for {
    header, err := tr.Next()
    if header.Typeflag == tar.TypeReg {
        content, _ := io.ReadAll(tr)
        dn.files[header.Name] = &dataFile{
            name:    filepath.Base(header.Name),
            content: content,
            modTime: header.ModTime,
        }
    }
}
```

### 5.3 Compressed Variants

```go
// gzip compressed
tarGz, err := node.ToTarGz()
node, err := datanode.FromTarGz(tarGzBytes)

// xz compressed
tarXz, err := node.ToTarXz()
node, err := datanode.FromTarXz(tarXzBytes)
```

## 6. File Modes

| Context | Mode | Notes |
|---------|------|-------|
| File read (fs.FS) | 0444 | Read-only for all |
| Directory (fs.FS) | 0555 | Read+execute for all |
| Tar export | 0600 | Owner read/write only |

**Note**: Original file modes are NOT preserved. All files get fixed modes.

## 7. Memory Model

- All content held in memory as `[]byte`
- No lazy loading
- No memory mapping
- Thread-safe for concurrent reads (map is not mutated after creation)

### 7.1 Size Calculation

```go
func (dn *DataNode) Size() int64 {
    var total int64
    for _, f := range dn.files {
        total += int64(len(f.content))
    }
    return total
}
```

## 8. Integration Points

### 8.1 TIM RootFS

```go
tim := &tim.TIM{
    Config: configJSON,
    RootFS: datanode,  // DataNode as container filesystem
}
```

### 8.2 TRIX Encryption

```go
// Encrypt DataNode to TRIX
encrypted, err := trix.Encrypt(datanode.ToTar(), password)

// Decrypt TRIX to DataNode
tarBytes, err := trix.Decrypt(encrypted, password)
node, err := datanode.FromTar(tarBytes)
```

### 8.3 Collectors

```go
// GitHub collector returns DataNode
node, err := github.CollectRepo(url)

// Website collector returns DataNode
node, err := website.Collect(url, depth)
```

## 9. Implementation Reference

- Source: `pkg/datanode/datanode.go`
- Tests: `pkg/datanode/datanode_test.go`

## 10. Security Considerations

1. **Path traversal**: Leading slashes stripped; no `..` handling needed (flat map)
2. **Memory exhaustion**: No built-in limits; caller must validate input size
3. **Tar bombs**: FromTar reads all entries into memory
4. **Symlinks**: Not supported (intentional - tar.TypeReg only)

## 11. Limitations

- No symlink support
- No extended attributes
- No sparse files
- Fixed file modes (0600 on export)
- No streaming (full content in memory)

## 12. Future Work

- [ ] Streaming tar generation for large files
- [ ] Optional mode preservation
- [ ] Size limits for untrusted input
- [ ] Lazy loading for large datasets

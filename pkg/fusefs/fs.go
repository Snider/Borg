package fusefs

import (
	"context"
	"io"
	"io/fs"
	"syscall"

	"github.com/Snider/Borg/pkg/datanode"
	fusefs "github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// DataNodeFs is a FUSE filesystem that serves a DataNode.
type DataNodeFs struct {
	fusefs.Inode
	dn *datanode.DataNode
}

// NewDataNodeFs creates a new DataNodeFs.
func NewDataNodeFs(dn *datanode.DataNode) *DataNodeFs {
	return &DataNodeFs{
		dn: dn,
	}
}

// Ensure we satisfy the FUSE interfaces.
var _ = (fusefs.NodeGetattrer)((*DataNodeFs)(nil))
var _ = (fusefs.NodeLookuper)((*DataNodeFs)(nil))
var _ = (fusefs.NodeReaddirer)((*DataNodeFs)(nil))
var _ = (fusefs.NodeOpener)((*DataNodeFs)(nil))

// fileHandle represents an open file in the FUSE filesystem.
type fileHandle struct {
	f fs.File
}

// Ensure we satisfy the FUSE file interfaces.
var _ = (fusefs.FileReader)((*fileHandle)(nil))
var _ = (fusefs.FileReleaser)((*fileHandle)(nil))


// Getattr gets the attributes of a file or directory.
func (r *DataNodeFs) Getattr(ctx context.Context, f fusefs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	info, err := r.dn.Stat(r.Path(&r.Inode))
	if err != nil {
		return syscall.ENOENT
	}
	out.Size = uint64(info.Size())
	out.Mode = uint32(info.Mode())
	out.Mtime = uint64(info.ModTime().Unix())
	return 0
}

// Lookup looks up a file or directory.
func (r *DataNodeFs) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fusefs.Inode, syscall.Errno) {
	path := r.Path(&r.Inode)
	if path == "." {
		path = ""
	}
	if path != "" {
		path += "/"
	}
	path += name

	info, err := r.dn.Stat(path)
	if err != nil {
		return nil, syscall.ENOENT
	}

	out.Size = uint64(info.Size())
	out.Mode = uint32(info.Mode())
	out.Mtime = uint64(info.ModTime().Unix())


	stable := fusefs.StableAttr{}
	if info.IsDir() {
		stable.Mode = fuse.S_IFDIR
	} else {
		stable.Mode = fuse.S_IFREG
	}

	node := &DataNodeFs{dn: r.dn}
	return r.NewInode(ctx, node, stable), 0
}

// Readdir lists the contents of a directory.
func (r *DataNodeFs) Readdir(ctx context.Context) (fusefs.DirStream, syscall.Errno) {
	path := r.Path(&r.Inode)
	if path == "" {
		path = "."
	}
	entries, err := r.dn.ReadDir(path)
	if err != nil {
		return nil, syscall.EIO
	}

	var result []fuse.DirEntry
	for _, entry := range entries {
		var mode uint32
		if entry.IsDir() {
			mode = fuse.S_IFDIR
		} else {
			mode = fuse.S_IFREG
		}
		result = append(result, fuse.DirEntry{
			Name: entry.Name(),
			Mode: mode,
		})
	}
	return fusefs.NewListDirStream(result), 0
}

// Open opens a file.
func (r *DataNodeFs) Open(ctx context.Context, flags uint32) (fusefs.FileHandle, uint32, syscall.Errno) {
	f, err := r.dn.Open(r.Path(&r.Inode))
	if err != nil {
		return nil, 0, syscall.ENOENT
	}
	return &fileHandle{f: f}, fuse.FOPEN_KEEP_CACHE, 0
}

// Read reads data from a file.
func (fh *fileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	r, ok := fh.f.(io.ReaderAt)
	if !ok {
		// Fallback for non-ReaderAt files
		if off == 0 {
			n, err := fh.f.Read(dest)
			if err != nil && err != io.EOF {
				return nil, syscall.EIO
			}
			return fuse.ReadResultData(dest[:n]), 0
		}
		return nil, syscall.EIO
	}
	n, err := r.ReadAt(dest, off)
	if err != nil && err != io.EOF {
		return nil, syscall.EIO
	}
	return fuse.ReadResultData(dest[:n]), 0
}

// Release closes an open file.
func (fh *fileHandle) Release(ctx context.Context) syscall.Errno {
	fh.f.(io.Closer).Close()
	return 0
}

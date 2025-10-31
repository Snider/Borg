package cmd

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve [file]",
	Short: "Serve a packaged PWA file",
	Long:  `Serves the contents of a packaged PWA file using a static file server.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pwaFile := args[0]
		port, _ := cmd.Flags().GetString("port")

		pwaData, err := os.ReadFile(pwaFile)
		if err != nil {
			fmt.Printf("Error reading PWA file: %v\n", err)
			return
		}

		memFS, err := newMemoryFS(pwaData)
		if err != nil {
			fmt.Printf("Error creating in-memory filesystem: %v\n", err)
			return
		}

		http.Handle("/", http.FileServer(http.FS(memFS)))

		fmt.Printf("Serving PWA on http://localhost:%s\n", port)
		err = http.ListenAndServe(":"+port, nil)
		if err != nil {
			fmt.Printf("Error starting server: %v\n", err)
			return
		}
	},
}

// memoryFS is an in-memory filesystem that implements fs.FS
type memoryFS struct {
	files map[string]*memoryFile
}

func newMemoryFS(tarball []byte) (*memoryFS, error) {
	memFS := &memoryFS{files: make(map[string]*memoryFile)}
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
			name := strings.TrimPrefix(header.Name, "/")
			memFS.files[name] = &memoryFile{
				name:    name,
				content: data,
				modTime: header.ModTime,
			}
		}
	}

	return memFS, nil
}

func (m *memoryFS) Open(name string) (fs.File, error) {
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		name = "index.html"
	}
	if file, ok := m.files[name]; ok {
		return &memoryFileReader{file: file}, nil
	}
	return nil, fs.ErrNotExist
}

// memoryFile represents a file in the in-memory filesystem
type memoryFile struct {
	name    string
	content []byte
	modTime time.Time
}

func (m *memoryFile) Stat() (fs.FileInfo, error) {
	return &memoryFileInfo{file: m}, nil
}

func (m *memoryFile) Read(p []byte) (int, error) {
	return 0, nil // This is implemented by memoryFileReader
}

func (m *memoryFile) Close() error {
	return nil
}

// memoryFileInfo implements fs.FileInfo for a memoryFile
type memoryFileInfo struct {
	file *memoryFile
}

func (m *memoryFileInfo) Name() string {
	return path.Base(m.file.name)
}

func (m *memoryFileInfo) Size() int64 {
	return int64(len(m.file.content))
}

func (m *memoryFileInfo) Mode() fs.FileMode {
	return 0444
}

func (m *memoryFileInfo) ModTime() time.Time {
	return m.file.modTime
}

func (m *memoryFileInfo) IsDir() bool {
	return false
}

func (m *memoryFileInfo) Sys() interface{} {
	return nil
}

// memoryFileReader implements fs.File for a memoryFile
type memoryFileReader struct {
	file   *memoryFile
	reader *bytes.Reader
}

func (m *memoryFileReader) Stat() (fs.FileInfo, error) {
	return m.file.Stat()
}

func (m *memoryFileReader) Read(p []byte) (int, error) {
	if m.reader == nil {
		m.reader = bytes.NewReader(m.file.content)
	}
	return m.reader.Read(p)
}

func (m *memoryFileReader) Close() error {
	return nil
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.PersistentFlags().String("port", "8080", "Port to serve the PWA on")
}

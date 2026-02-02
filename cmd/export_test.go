package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"github.com/alexmullins/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportCmd_Dir(t *testing.T) {
	// Create a test datanode
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello"))
	dn.AddData("dir/file2.txt", []byte("world"))

	trixData, err := trix.ToTrix(dn, "")
	require.NoError(t, err)

	// Write the datanode to a temporary file
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.dat")
	require.NoError(t, err)
	_, err = tmpFile.Write(trixData)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a temporary output directory
	outDir := t.TempDir()

	// Execute the export command
	cmd := NewRootCmd()
	cmd.AddCommand(GetExportCmd())
	cmd.SetArgs([]string{"export", tmpFile.Name(), "--format", "dir", "-o", outDir})
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify the output
	assert.FileExists(t, filepath.Join(outDir, "file1.txt"))
	assert.FileExists(t, filepath.Join(outDir, "dir/file2.txt"))

	content1, err := os.ReadFile(filepath.Join(outDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content1))

	content2, err := os.ReadFile(filepath.Join(outDir, "dir/file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "world", string(content2))
}

func TestExportCmd_Zip(t *testing.T) {
	// Create a test datanode
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello"))
	dn.AddData("dir/file2.txt", []byte("world"))

	trixData, err := trix.ToTrix(dn, "")
	require.NoError(t, err)

	// Write the datanode to a temporary file
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.dat")
	require.NoError(t, err)
	_, err = tmpFile.Write(trixData)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a temporary output file
	outZip := filepath.Join(t.TempDir(), "out.zip")

	// Execute the export command
	cmd := NewRootCmd()
	cmd.AddCommand(GetExportCmd())
	cmd.SetArgs([]string{"export", tmpFile.Name(), "--format", "zip", "-o", outZip})
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify the output
	zipReader, err := zip.OpenReader(outZip)
	require.NoError(t, err)
	defer zipReader.Close()

	found1 := false
	found2 := false
	for _, f := range zipReader.File {
		if f.Name == "file1.txt" {
			found1 = true
			rc, err := f.Open()
			require.NoError(t, err)
			defer rc.Close()
			content, err := io.ReadAll(rc)
			require.NoError(t, err)
			assert.Equal(t, "hello", string(content))
		}
		if f.Name == "dir/file2.txt" {
			found2 = true
			rc, err := f.Open()
			require.NoError(t, err)
			defer rc.Close()
			content, err := io.ReadAll(rc)
			require.NoError(t, err)
			assert.Equal(t, "world", string(content))
		}
	}
	assert.True(t, found1, "file1.txt not found in zip")
	assert.True(t, found2, "dir/file2.txt not found in zip")
}

func TestExportCmd_TarGz(t *testing.T) {
	// Create a test datanode
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello"))
	dn.AddData("dir/file2.txt", []byte("world"))

	trixData, err := trix.ToTrix(dn, "")
	require.NoError(t, err)

	// Write the datanode to a temporary file
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.dat")
	require.NoError(t, err)
	_, err = tmpFile.Write(trixData)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a temporary output file
	outTarGz := filepath.Join(t.TempDir(), "out.tar.gz")

	// Execute the export command
	cmd := NewRootCmd()
	cmd.AddCommand(GetExportCmd())
	cmd.SetArgs([]string{"export", tmpFile.Name(), "--format", "tar.gz", "-o", outTarGz})
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify the output
	file, err := os.Open(outTarGz)
	require.NoError(t, err)
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	found1 := false
	found2 := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if header.Name == "file1.txt" {
			found1 = true
			content, err := io.ReadAll(tarReader)
			require.NoError(t, err)
			assert.Equal(t, "hello", string(content))
		}
		if header.Name == "dir/file2.txt" {
			found2 = true
			content, err := io.ReadAll(tarReader)
			require.NoError(t, err)
			assert.Equal(t, "world", string(content))
		}
	}
	assert.True(t, found1, "file1.txt not found in tar.gz")
	assert.True(t, found2, "dir/file2.txt not found in tar.gz")
}

func TestExportCmd_InvalidFormat(t *testing.T) {
	dn := datanode.New()
	trixData, err := trix.ToTrix(dn, "")
	require.NoError(t, err)

	// Create a temporary file
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.dat")
	require.NoError(t, err)
	_, err = tmpFile.Write(trixData)
	require.NoError(t, err)
	tmpFile.Close()

	// Execute the export command with an invalid format
	cmd := NewRootCmd()
	cmd.AddCommand(GetExportCmd())
	cmd.SetArgs([]string{"export", tmpFile.Name(), "--format", "invalid", "-o", "out"})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format: invalid")
}

func TestExportCmd_Filtering(t *testing.T) {
	// Create a test datanode
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello"))
	dn.AddData("dir/file2.txt", []byte("world"))
	dn.AddData("dir/image.jpg", []byte("world"))

	trixData, err := trix.ToTrix(dn, "")
	require.NoError(t, err)

	// Write the datanode to a temporary file
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.dat")
	require.NoError(t, err)
	_, err = tmpFile.Write(trixData)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a temporary output directory
	outDir := t.TempDir()

	// Execute the export command
	cmd := NewRootCmd()
	cmd.AddCommand(GetExportCmd())
	cmd.SetArgs([]string{"export", tmpFile.Name(), "--format", "dir", "-o", outDir, "--include", "*.txt", "--exclude", "dir/*"})
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify the output
	assert.FileExists(t, filepath.Join(outDir, "file1.txt"))
	assert.NoFileExists(t, filepath.Join(outDir, "dir/file2.txt"))
	assert.NoFileExists(t, filepath.Join(outDir, "dir/image.jpg"))
}

func TestExportCmd_ZipEncryption(t *testing.T) {
	// Create a test datanode
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello"))

	// Create a .stim file
	m, err := tim.FromDataNode(dn)
	require.NoError(t, err)
	stimData, err := m.ToSigil("password")
	require.NoError(t, err)

	// Write the datanode to a temporary file
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.stim")
	require.NoError(t, err)
	_, err = tmpFile.Write(stimData)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a temporary output file
	outZip := filepath.Join(t.TempDir(), "out.zip")

	// Execute the export command
	cmd := NewRootCmd()
	cmd.AddCommand(GetExportCmd())
	cmd.SetArgs([]string{"export", tmpFile.Name(), "--format", "zip", "-o", outZip, "-p", "password"})
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify the output
	zipReader, err := zip.OpenReader(outZip)
	require.NoError(t, err)
	defer zipReader.Close()

	assert.Len(t, zipReader.File, 1)
	f := zipReader.File[0]
	assert.Equal(t, "file1.txt", f.Name)

	f.SetPassword("password")
	rc, err := f.Open()
	require.NoError(t, err)
	defer rc.Close()
	content, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))
}

func TestExportCmd_StimInput(t *testing.T) {
	// Create a test datanode
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello"))

	// Create a .stim file
	m, err := tim.FromDataNode(dn)
	require.NoError(t, err)
	stimData, err := m.ToSigil("password")
	require.NoError(t, err)

	// Write the datanode to a temporary file
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.stim")
	require.NoError(t, err)
	_, err = tmpFile.Write(stimData)
	require.NoError(t, err)
	tmpFile.Close()

	// Create a temporary output directory
	outDir := t.TempDir()

	// Execute the export command
	cmd := NewRootCmd()
	cmd.AddCommand(GetExportCmd())
	cmd.SetArgs([]string{"export", tmpFile.Name(), "--format", "dir", "-o", outDir, "-p", "password"})
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify the output
	assert.FileExists(t, filepath.Join(outDir, "file1.txt"))
	content, err := os.ReadFile(filepath.Join(outDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))
}

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/stretchr/testify/assert"
)

func TestVerifyCmd_Good(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test.dat")

	// Create a DataNode and add a file
	dn := datanode.New()
	dn.AddData("hello.txt", []byte("hello world"))

	// Serialize the DataNode to a tarball
	tarball, err := dn.ToTar()
	assert.NoError(t, err)

	// Write the tarball to the archive file
	err = os.WriteFile(archivePath, tarball, 0644)
	assert.NoError(t, err)

	// Execute the verify command
	cmd := NewVerifyCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{archivePath})
	err = cmd.Execute()

	// Assert that the command was successful
	assert.NoError(t, err)
	output := b.String()
	assert.Contains(t, output, "✓ Structure: valid")
	assert.Contains(t, output, "✓ Checksums: 1/1 files OK")
	assert.Contains(t, output, "✓ Manifest: complete")
}

func TestVerifyCmd_Bad(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test.dat")

	// Create a DataNode and add a file
	dn := datanode.New()
	dn.AddData("hello.txt", []byte("hello world"))

	// Serialize the DataNode to a tarball
	tarball, err := dn.ToTar()
	assert.NoError(t, err)

	// Corrupt the tarball
	corruptedTarball := bytes.Replace(tarball, []byte("hello world"), []byte("hello mars!"), 1)

	// Write the corrupted tarball to the archive file
	err = os.WriteFile(archivePath, corruptedTarball, 0644)
	assert.NoError(t, err)

	// Execute the verify command
	cmd := NewVerifyCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{archivePath})
	err = cmd.Execute()

	// Assert that the command failed
	assert.Error(t, err)
	output := b.String()
	assert.Contains(t, output, "✓ Structure: valid")
	assert.Contains(t, output, "✗ Checksum mismatch: hello.txt")
	assert.Contains(t, output, "✗ Checksums: 0/1 files OK")
	assert.Contains(t, output, "✗ Manifest: incomplete")
}

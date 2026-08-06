package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/stretchr/testify/assert"
)

func TestManifestCmd_Good(t *testing.T) {
	// Create a test archive
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello"))
	dn.AddData("file2.txt", []byte("world"))
	tarball, err := dn.ToTar()
	assert.NoError(t, err)

	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test.dat")
	err = os.WriteFile(archivePath, tarball, 0644)
	assert.NoError(t, err)

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetManifestCmd())

	output, err := executeCommand(rootCmd, "manifest", archivePath)
	assert.NoError(t, err)

	// Verify output
	assert.Contains(t, output, `"total_files": 2`)
	assert.Contains(t, output, `"total_size": "10 B"`)
}

package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/stretchr/testify/assert"
)

func TestRepairCmd_Good(t *testing.T) {
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

	// Create a mock HTTP server to serve the correct file
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello.txt" {
			w.Write([]byte("hello world"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Execute the repair command
	cmd := NewRepairCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{archivePath, "--source", server.URL})
	err = cmd.Execute()

	// Assert that the command was successful
	assert.NoError(t, err)

	// Verify that the archive is now valid
	verifyCmd := NewVerifyCmd()
	b.Reset()
	verifyCmd.SetOut(b)
	verifyCmd.SetArgs([]string{archivePath})
	err = verifyCmd.Execute()
	assert.NoError(t, err)
	output := b.String()
	assert.Contains(t, output, "✓ Structure: valid")
	assert.Contains(t, output, "✓ Checksums: 1/1 files OK")
	assert.Contains(t, output, "✓ Manifest: complete")
}

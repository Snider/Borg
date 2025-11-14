package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/trix"
)

func TestDecodeCmd(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "decoded.dat")
		inputFile := filepath.Join(tempDir, "test.trix")

		// Create a dummy trix file.
		dn := datanode.New()
		dn.AddData("test.txt", []byte("hello"))
		trixBytes, err := trix.ToTrix(dn, "")
		if err != nil {
			t.Fatalf("failed to create trix file: %v", err)
		}
		err = os.WriteFile(inputFile, trixBytes, 0644)
		if err != nil {
			t.Fatalf("failed to write trix file: %v", err)
		}

		// Execute the decode command.
		cmd := NewDecodeCmd()
		cmd.SetArgs([]string{inputFile, "--output", outputFile})
		err = cmd.Execute()
		if err != nil {
			t.Fatalf("decode command failed: %v", err)
		}

		// Verify the output file.
		_, err = os.Stat(outputFile)
		if err != nil {
			t.Fatalf("output file not found: %v", err)
		}
	})
}

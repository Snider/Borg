package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Snider/Enchantrix/pkg/trix"
	"github.com/stretchr/testify/assert"
)

func TestCompileCmd(t *testing.T) {
	// t.Run("Good", func(t *testing.T) {
	// 	tempDir := t.TempDir()
	// 	outputTimPath := filepath.Join(tempDir, "test.tim")
	// 	borgfilePath := filepath.Join(tempDir, "Borgfile")
	// 	dummyFilePath := filepath.Join(tempDir, "dummy.txt")

	// 	// Create a dummy file to add to the tim.
	// 	err := os.WriteFile(dummyFilePath, []byte("dummy content"), 0644)
	// 	if err != nil {
	// 		t.Fatalf("failed to create dummy file: %v", err)
	// 	}

	// 	// Create a Borgfile.
	// 	borgfileContent := "ADD " + dummyFilePath + " /dummy.txt"
	// 	err = os.WriteFile(borgfilePath, []byte(borgfileContent), 0644)
	// 	if err != nil {
	// 		t.Fatalf("failed to create Borgfile: %v", err)
	// 	}

	// 	// Execute the compile command.
	// 	cmd := NewCompileCmd()
	// 	cmd.SetArgs([]string{"--file", borgfilePath, "--output", outputTimPath})
	// 	err = cmd.Execute()
	// 	if err != nil {
	// 		t.Fatalf("compile command failed: %v", err)
	// 	}

	// 	// Verify the output tim file.
	// 	timFile, err := os.Open(outputTimPath)
	// 	if err != nil {
	// 		t.Fatalf("failed to open output tim file: %v", err)
	// 	}
	// 	defer timFile.Close()

	// 	tr := tar.NewReader(timFile)
	// 	files := []string{"config.json", "rootfs/", "rootfs/dummy.txt"}
	// 	found := make(map[string]bool)
	// 	for {
	// 		hdr, err := tr.Next()
	// 		if err != nil {
	// 			break
	// 		}
	// 		found[hdr.Name] = true
	// 	}
	// 	for _, f := range files {
	// 		if !found[f] {
	// 			t.Errorf("%s not found in tim tarball", f)
	// 		}
	// 	}
	// })

	t.Run("Bad_Borgfile", func(t *testing.T) {
		tempDir := t.TempDir()
		outputTimPath := filepath.Join(tempDir, "test.tim")
		borgfilePath := filepath.Join(tempDir, "Borgfile")

		// Create a Borgfile with an invalid instruction.
		borgfileContent := "INVALID instruction"
		err := os.WriteFile(borgfilePath, []byte(borgfileContent), 0644)
		if err != nil {
			t.Fatalf("failed to create Borgfile: %v", err)
		}

		// Execute the compile command.
		cmd := NewCompileCmd()
		cmd.SetArgs([]string{"--file", borgfilePath, "--output", outputTimPath})
		err = cmd.Execute()
		if err == nil {
			t.Error("compile command should have failed but did not")
		}
	})

	t.Run("Bad_ADD", func(t *testing.T) {
		tempDir := t.TempDir()
		outputTimPath := filepath.Join(tempDir, "test.tim")
		borgfilePath := filepath.Join(tempDir, "Borgfile")

		// Create a Borgfile with an invalid ADD instruction.
		borgfileContent := "ADD dummy.txt"
		err := os.WriteFile(borgfilePath, []byte(borgfileContent), 0644)
		if err != nil {
			t.Fatalf("failed to create Borgfile: %v", err)
		}

		// Execute the compile command.
		cmd := NewCompileCmd()
		cmd.SetArgs([]string{"--file", borgfilePath, "--output", outputTimPath})
		err = cmd.Execute()
		if err == nil {
			t.Error("compile command should have failed but did not")
		}
	})

	t.Run("Ugly_EmptyBorgfile", func(t *testing.T) {
		tempDir := t.TempDir()
		outputTimPath := filepath.Join(tempDir, "test.tim")
		borgfilePath := filepath.Join(tempDir, "Borgfile")

		// Create an empty Borgfile.
		err := os.WriteFile(borgfilePath, []byte{}, 0644)
		if err != nil {
			t.Fatalf("failed to create Borgfile: %v", err)
		}

		// Execute the compile command.
		cmd := NewCompileCmd()
		cmd.SetArgs([]string{"--file", borgfilePath, "--output", outputTimPath})
		err = cmd.Execute()
		if err != nil {
			t.Fatalf("compile command failed: %v", err)
		}
	})
}

func TestCompileCmd_WithPublicManifest_Good(t *testing.T) {
	tempDir := t.TempDir()
	outputStimPath := filepath.Join(tempDir, "test.stim")
	borgfilePath := filepath.Join(tempDir, "Borgfile")
	dummyFilePath := filepath.Join(tempDir, "dummy.txt")

	// Create a dummy file to add to the tim.
	err := os.WriteFile(dummyFilePath, []byte("dummy content"), 0644)
	assert.NoError(t, err)

	// Create a Borgfile.
	borgfileContent := "ADD " + dummyFilePath + " /dummy.txt"
	err = os.WriteFile(borgfilePath, []byte(borgfileContent), 0644)
	assert.NoError(t, err)

	// Execute the compile command.
	cmd := NewCompileCmd()
	cmd.SetArgs([]string{"--file", borgfilePath, "--output", outputStimPath, "--encrypt", "password", "--public-manifest"})
	err = cmd.Execute()
	assert.NoError(t, err)

	// Verify the output stim file.
	data, err := os.ReadFile(outputStimPath)
	assert.NoError(t, err)

	decodedTrix, err := trix.Decode(data, "STIM", nil)
	assert.NoError(t, err)
	assert.NotNil(t, decodedTrix)

	manifest, ok := decodedTrix.Header["public_manifest"].(string)
	assert.True(t, ok)
	assert.Contains(t, manifest, `"total_files": 1`)
}

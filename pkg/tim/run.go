package tim

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ExecCommand = exec.Command
)

func Run(timPath string) error {
	// Create a temporary directory to unpack the tim file.
	tempDir, err := os.MkdirTemp("", "borg-run-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Unpack the tim file.
	file, err := os.Open(timPath)
	if err != nil {
		return fmt.Errorf("failed to open tim file: %w", err)
	}
	defer file.Close()

	tr := tar.NewReader(file)
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}

		target := filepath.Join(tempDir, hdr.Name)
		target = filepath.Clean(target)
		if !strings.HasPrefix(target, filepath.Clean(tempDir)+string(os.PathSeparator)) && target != filepath.Clean(tempDir) {
			return fmt.Errorf("invalid file path: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			if err := outFile.Close(); err != nil {
				return fmt.Errorf("failed to close file: %w", err)
			}
		}
	}

	// Run the tim.
	cmd := ExecCommand("runc", "run", "-b", tempDir, "borg-container")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunEncrypted runs an encrypted .stim file.
// It decrypts the file, extracts the contents to a temporary directory,
// and runs the container using runc.
func RunEncrypted(stimPath, password string) error {
	// Read the encrypted file
	data, err := os.ReadFile(stimPath)
	if err != nil {
		return fmt.Errorf("failed to read stim file: %w", err)
	}

	// Decrypt and parse
	m, err := FromSigil(data, password)
	if err != nil {
		return fmt.Errorf("failed to decrypt stim: %w", err)
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "borg-run-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write config.json
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, m.Config, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Extract rootfs
	rootfsPath := filepath.Join(tempDir, "rootfs")
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		return fmt.Errorf("failed to create rootfs dir: %w", err)
	}

	// Walk DataNode and extract files
	err = m.RootFS.Walk(".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		target := filepath.Join(rootfsPath, path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return m.RootFS.CopyFile(path, target, 0600)
	})
	if err != nil {
		return fmt.Errorf("failed to extract rootfs: %w", err)
	}

	// Run with runc
	cmd := ExecCommand("runc", "run", "-b", tempDir, "borg-container")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

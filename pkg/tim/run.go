package tim

import (
	"archive/tar"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer outFile.Close()
			if _, err := outFile.ReadFrom(tr); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
		}
	}

	// Run the tim.
	cmd := ExecCommand("runc", "run", "-b", tempDir, "borg-container")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

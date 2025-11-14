package tim

import (
	"archive/tar"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// ExecCommand is a function variable that creates a new exec.Cmd. It is a
// variable to allow for mocking in tests.
var ExecCommand = exec.Command

// Run unpacks and executes a TIM from a given tarball path
// using runc. It unpacks the bundle into a temporary directory, then executes it
// with "runc run".
//
// Note: This function requires "runc" to be installed and in the system's PATH.
func Run(timPath string) error {
	// Create a temporary directory to unpack the TIM file.
	tempDir, err := os.MkdirTemp("", "borg-run-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Unpack the TIM file.
	file, err := os.Open(timPath)
	if err != nil {
		return err
	}
	defer file.Close()

	tr := tar.NewReader(file)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(tempDir, header.Name)
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
			continue
		}

		outFile, err := os.Create(path)
		if err != nil {
			return err
		}
		defer outFile.Close()
		if _, err := io.Copy(outFile, tr); err != nil {
			return err
		}
	}

	// Run the TIM.
	runc := ExecCommand("runc", "run", "borg-container")
	runc.Dir = tempDir
	runc.Stdout = os.Stdout
	runc.Stderr = os.Stderr
	runc.Stdin = os.Stdin
	return runc.Run()
}

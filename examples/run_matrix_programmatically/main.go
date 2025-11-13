package main

import (
	"archive/tar"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Open the matrix file.
	matrixFile, err := os.Open("programmatic.matrix")
	if err != nil {
		log.Fatalf("Failed to open matrix file (run create_matrix_programmatically first): %v", err)
	}
	defer matrixFile.Close()

	// Create a temporary directory to unpack the matrix.
	tempDir, err := os.MkdirTemp("", "borg-run-example-*")
	if err != nil {
		log.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	log.Printf("Unpacking matrix to %s", tempDir)

	// Unpack the tar archive.
	tr := tar.NewReader(matrixFile)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatalf("Failed to read tar header: %v", err)
		}

		target := filepath.Join(tempDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				log.Fatalf("Failed to create directory: %v", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				log.Fatalf("Failed to create file: %v", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				log.Fatalf("Failed to write file content: %v", err)
			}
			outFile.Close()
		default:
			log.Printf("Skipping unsupported type: %c in %s", header.Typeflag, header.Name)
		}
	}

	log.Println("Executing matrix with runc...")

	// Execute the matrix using runc.
	cmd := exec.Command("runc", "run", "borg-container-example")
	cmd.Dir = tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run matrix: %v", err)
	}

	log.Println("Matrix execution finished.")
}

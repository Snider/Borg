package cmd

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var runCmd = NewRunCmd()

func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [matrix file]",
		Short: "Run a Terminal Isolation Matrix.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			matrixFile := args[0]

			// Create a temporary directory to unpack the matrix file.
			tempDir, err := os.MkdirTemp("", "borg-run-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tempDir)

			// Unpack the matrix file.
			file, err := os.Open(matrixFile)
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

			// Run the matrix.
			runc := execCommand("runc", "run", "borg-container")
			runc.Dir = tempDir
			runc.Stdout = os.Stdout
			runc.Stderr = os.Stderr
			runc.Stdin = os.Stdin
			return runc.Run()
		},
	}
}

func GetRunCmd() *cobra.Command {
	return runCmd
}

func init() {
	RootCmd.AddCommand(GetRunCmd())
}

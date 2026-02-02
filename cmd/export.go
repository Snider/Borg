package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	azip "github.com/alexmullins/zip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/spf13/cobra"
)

// shouldInclude determines if a file path should be included based on include and exclude patterns.
// Exclude patterns take precedence. If a path matches an exclude pattern, it's excluded.
// If include patterns are provided, the path must match at least one of them.
// If no include patterns are provided, all paths are included by default (unless excluded).
func shouldInclude(path string, include, exclude []string) (bool, error) {
	for _, pattern := range exclude {
		if matched, err := filepath.Match(pattern, path); err != nil {
			return false, fmt.Errorf("error matching exclude pattern '%s': %w", pattern, err)
		} else if matched {
			return false, nil
		}
	}

	if len(include) > 0 {
		for _, pattern := range include {
			if matched, err := filepath.Match(pattern, path); err != nil {
				return false, fmt.Errorf("error matching include pattern '%s': %w", pattern, err)
			} else if matched {
				return true, nil
			}
		}
		return false, nil // Must match an include pattern if provided
	}

	return true, nil // Include by default if no include patterns
}

var exportCmd = NewExportCmd()

func NewExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [file]",
		Short: "Export an archive to a different format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputFile := args[0]
			output, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			password, _ := cmd.Flags().GetString("password")
			include, _ := cmd.Flags().GetStringSlice("include")
			exclude, _ := cmd.Flags().GetStringSlice("exclude")

			data, err := os.ReadFile(inputFile)
			if err != nil {
				return fmt.Errorf("failed to read input file: %w", err)
			}

			var dn *datanode.DataNode

			// Handle .stim (encrypted TIM)
			if strings.HasSuffix(inputFile, ".stim") || (len(data) > 4 && string(data[:4]) == "STIM") {
				if password == "" {
					return fmt.Errorf("password required for .stim files")
				}
				m, err := tim.FromSigil(data, password)
				if err != nil {
					return fmt.Errorf("failed to decode .stim file: %w", err)
				}
				dn = m.RootFS
			} else {
				// Handle .dat, .trix, .tim
				dn, err = trix.FromTrix(data, password)
				if err != nil {
					return fmt.Errorf("failed to decode archive: %w", err)
				}
			}

			switch format {
			case "dir":
				err := os.MkdirAll(output, 0755)
				if err != nil {
					return fmt.Errorf("failed to create output directory: %w", err)
				}
				err = dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if path == "." {
						return nil
					} // Skip root

					includePath, err := shouldInclude(path, include, exclude)
					if err != nil {
						return err
					}

					if d.IsDir() {
						if !includePath {
							return fs.SkipDir
						}
						return os.MkdirAll(filepath.Join(output, path), 0755)
					}

					if !includePath {
						return nil
					}
					return dn.CopyFile(path, filepath.Join(output, path), 0644)
				})
				if err != nil {
					return fmt.Errorf("failed to export to directory: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Exported to %s\n", output)
				return nil
			case "zip":
				zipFile, err := os.Create(output)
				if err != nil {
					return fmt.Errorf("failed to create zip file: %w", err)
				}
				defer zipFile.Close()

				zipWriter := azip.NewWriter(zipFile)
				defer zipWriter.Close()

				err = dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if path == "." {
						return nil
					}

					includePath, err := shouldInclude(path, include, exclude)
					if err != nil {
						return err
					}

					if d.IsDir() {
						if !includePath {
							return fs.SkipDir
						}
						_, err := zipWriter.Create(path + "/")
						return err
					}

					if !includePath {
						return nil
					}

					var writer io.Writer
					if password != "" {
						writer, err = zipWriter.Encrypt(path, password)
					} else {
						writer, err = zipWriter.Create(path)
					}
					if err != nil {
						return err
					}

					file, err := dn.Open(path)
					if err != nil {
						return err
					}
					defer file.Close()
					_, err = io.Copy(writer, file)
					return err
				})
				if err != nil {
					return fmt.Errorf("failed to export to zip: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Exported to %s\n", output)
				return nil
			case "tar.gz":
				tarFile, err := os.Create(output)
				if err != nil {
					return fmt.Errorf("failed to create tar.gz file: %w", err)
				}
				defer tarFile.Close()

				gzipWriter := gzip.NewWriter(tarFile)
				defer gzipWriter.Close()

				tarWriter := tar.NewWriter(gzipWriter)
				defer tarWriter.Close()

				err = dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if path == "." {
						return nil
					}

					includePath, err := shouldInclude(path, include, exclude)
					if err != nil {
						return err
					}

					if d.IsDir() {
						if !includePath {
							return fs.SkipDir
						}
					} else { // It's a file
						if !includePath {
							return nil
						}
					}

					info, err := d.Info()
					if err != nil {
						return err
					}
					header, err := tar.FileInfoHeader(info, "")
					if err != nil {
						return err
					}
					header.Name = path
					if err := tarWriter.WriteHeader(header); err != nil {
						return err
					}

					if !d.IsDir() {
						file, err := dn.Open(path)
						if err != nil {
							return err
						}
						defer file.Close()
						_, err = io.Copy(tarWriter, file)
						return err
					}

					return nil
				})
				if err != nil {
					return fmt.Errorf("failed to export to tar.gz: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Exported to %s\n", output)
				return nil
			default:
				return fmt.Errorf("unsupported format: %s", format)
			}
		},
	}

	cmd.Flags().StringP("format", "f", "zip", "Output format (zip, tar.gz, dir)")
	cmd.Flags().StringP("output", "o", "", "Output file or directory")
	cmd.Flags().StringP("password", "p", "", "Password for encryption (zip)")
	cmd.Flags().StringSlice("include", []string{}, "Patterns of files to include")
	cmd.Flags().StringSlice("exclude", []string{}, "Patterns of files to exclude")
	cmd.MarkFlagRequired("output")

	return cmd
}

func GetExportCmd() *cobra.Command {
	return exportCmd
}

func init() {
	RootCmd.AddCommand(GetExportCmd())
}

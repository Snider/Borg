package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/pdf"
	"github.com/spf13/cobra"
)

// extractMetadataCmd represents the extract-metadata command
var extractMetadataCmd = NewExtractMetadataCmd()

func init() {
	RootCmd.AddCommand(GetExtractMetadataCmd())
}

func NewExtractMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract-metadata [archive]",
		Short: "Extract metadata from files in an archive.",
		Long:  `Extract metadata from files of a specific type within a DataNode archive and create an INDEX.json file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			archivePath := args[0]
			fileType, _ := cmd.Flags().GetString("type")

			if fileType != "pdf" {
				return fmt.Errorf("unsupported type: %s. Only 'pdf' is currently supported", fileType)
			}

			// Read and decompress the archive
			compressedData, err := os.ReadFile(archivePath)
			if err != nil {
				return fmt.Errorf("failed to read archive file: %w", err)
			}
			data, err := compress.Decompress(compressedData)
			if err != nil {
				return fmt.Errorf("failed to decompress archive: %w", err)
			}

			// Load the DataNode
			dn, err := datanode.FromTar(data)
			if err != nil {
				return fmt.Errorf("failed to load DataNode from tar: %w", err)
			}

			var allMetadata []*pdf.Metadata

			// Walk the DataNode and extract metadata from PDF files
			err = dn.Walk("/", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".pdf") {
					// Create a temporary file to run extraction on
					tempFile, err := os.CreateTemp("", "borg-pdf-*.pdf")
					if err != nil {
						return fmt.Errorf("failed to create temp file: %w", err)
					}
					defer os.Remove(tempFile.Name())

					// Get the file content from DataNode
					file, err := dn.Open(path)
					if err != nil {
						return fmt.Errorf("failed to open %s from DataNode: %w", path, err)
					}
					defer file.Close()

					// Copy content to temp file
					if _, err := io.Copy(tempFile, file); err != nil {
						return fmt.Errorf("failed to copy content to temp file: %w", err)
					}
					tempFile.Close() // Close the file to allow reading by the extractor

					// Extract metadata
					metadata, err := pdf.ExtractMetadata(tempFile.Name())
					if err != nil {
						// Log error but continue processing other files
						fmt.Fprintf(cmd.ErrOrStderr(), "could not extract metadata from %s: %v\n", path, err)
						return nil
					}
					metadata.File = filepath.Base(path) // Use the original filename
					allMetadata = append(allMetadata, metadata)
				}
				return nil
			})

			if err != nil {
				return fmt.Errorf("error walking DataNode: %w", err)
			}

			// Write the aggregated metadata to INDEX.json
			jsonOutput, err := json.MarshalIndent(allMetadata, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal metadata to JSON: %w", err)
			}

			err = os.WriteFile("INDEX.json", jsonOutput, 0644)
			if err != nil {
				return fmt.Errorf("failed to write INDEX.json: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Metadata extracted and saved to INDEX.json")
			return nil
		},
	}
	cmd.Flags().String("type", "pdf", "The type of files to extract metadata from (currently only 'pdf' is supported)")
	return cmd
}

func GetExtractMetadataCmd() *cobra.Command {
	return extractMetadataCmd
}

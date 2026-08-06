package cmd

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/spf13/cobra"
)

// indexCmd represents the index command
var indexCmd = NewIndexCmd()

func init() {
	RootCmd.AddCommand(GetIndexCmd())
}

func NewIndexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "index <archive>",
		Short: "Build search index for an archive.",
		Long:  `Build a search index for a .dat, .tim, or .trix archive.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			archivePath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("failed to get absolute path for archive: %w", err)
			}

			// Read and decompress the archive
			compressedData, err := os.ReadFile(archivePath)
			if err != nil {
				return fmt.Errorf("failed to read archive: %w", err)
			}
			tarData, err := compress.Decompress(compressedData)
			if err != nil {
				return fmt.Errorf("failed to decompress archive: %w", err)
			}

			// Load the DataNode
			dn, err := datanode.FromTar(tarData)
			if err != nil {
				return fmt.Errorf("failed to load datanode: %w", err)
			}

			// Build the index
			trigramIndex := make(map[[3]byte][]uint32)
			var fileList []string

			err = dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}

				// Add file to list and map
				fileID := uint32(len(fileList))
				fileList = append(fileList, path)

				// Read file content
				file, err := dn.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				content, err := io.ReadAll(file)
				if err != nil {
					return err
				}

				// Generate and add trigrams
				if len(content) < 3 {
					return nil
				}
				for i := 0; i <= len(content)-3; i++ {
					var trigram [3]byte
					copy(trigram[:], content[i:i+3])

					postings := trigramIndex[trigram]
					if len(postings) == 0 || postings[len(postings)-1] != fileID {
						trigramIndex[trigram] = append(postings, fileID)
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to walk datanode: %w", err)
			}

			// Save the index
			indexDir := filepath.Join(filepath.Dir(archivePath), ".borg-index")
			if err := os.MkdirAll(indexDir, 0755); err != nil {
				return fmt.Errorf("failed to create index directory: %w", err)
			}

			// Save file list
			fileListPath := filepath.Join(indexDir, "files.json")
			fileListData, err := json.MarshalIndent(fileList, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal file list: %w", err)
			}
			if err := os.WriteFile(fileListPath, fileListData, 0644); err != nil {
				return fmt.Errorf("failed to write file list: %w", err)
			}

			// Save trigram index
			trigramIndexPath := filepath.Join(indexDir, "trigram.idx")
			var buf bytes.Buffer
			encoder := gob.NewEncoder(&buf)
			if err := encoder.Encode(trigramIndex); err != nil {
				return fmt.Errorf("failed to encode trigram index: %w", err)
			}
			if err := os.WriteFile(trigramIndexPath, buf.Bytes(), 0644); err != nil {
				return fmt.Errorf("failed to write trigram index: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Successfully built index for %s\n", args[0])
			return nil
		},
	}
}

func GetIndexCmd() *cobra.Command {
	return indexCmd
}

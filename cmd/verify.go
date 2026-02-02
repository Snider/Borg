package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/spf13/cobra"
)

// verifyCmd represents the verify command
var verifyCmd = NewVerifyCmd()

func init() {
	RootCmd.AddCommand(GetVerifyCmd())
}
func NewVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify [archive]",
		Short: "Verify archive integrity and detect corruption.",
		Long:  `Verify archive integrity and detect corruption.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			archivePath := args[0]
			password, _ := cmd.Flags().GetString("password")
			cmd.Printf("Verifying %s...\n", archivePath)

			content, err := os.ReadFile(archivePath)
			if err != nil {
				return fmt.Errorf("could not read archive: %w", err)
			}

			var dn *datanode.DataNode
			if strings.HasSuffix(archivePath, ".stim") {
				if password == "" {
					return fmt.Errorf("password required for .stim files")
				}
				t, err := tim.FromSigil(content, password)
				if err != nil {
					cmd.Println("✗ Decryption: failed")
					return fmt.Errorf("decryption failed: %w", err)
				}
				dn = t.RootFS
				cmd.Println("✓ Decryption: successful")
			} else {
				dn, err = datanode.FromTar(content)
				if err != nil {
					cmd.Println("✗ Structure: invalid")
					return fmt.Errorf("archive structure is corrupt: %w", err)
				}
				cmd.Println("✓ Structure: valid")
			}

			manifestFile, err := dn.Open("manifest.json")
			if err != nil {
				cmd.Println("✗ Checksums: missing manifest")
				return fmt.Errorf("could not open manifest: %w", err)
			}
			defer manifestFile.Close()

			manifestBytes, err := io.ReadAll(manifestFile)
			if err != nil {
				cmd.Println("✗ Checksums: unreadable manifest")
				return fmt.Errorf("could not read manifest: %w", err)
			}

			var manifest map[string]string
			if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
				cmd.Println("✗ Checksums: invalid manifest")
				return fmt.Errorf("could not parse manifest: %w", err)
			}

			filesOk := 0
			filesCorrupt := 0
			for name, expectedChecksum := range manifest {
				file, err := dn.Open(name)
				if err != nil {
					cmd.Printf("✗ Checksum mismatch: %s (missing)\n", name)
					filesCorrupt++
					continue
				}
				defer file.Close()

				fileBytes, err := io.ReadAll(file)
				if err != nil {
					cmd.Printf("✗ Checksum mismatch: %s (unreadable)\n", name)
					filesCorrupt++
					continue
				}

				hash := sha256.Sum256(fileBytes)
				actualChecksum := hex.EncodeToString(hash[:])

				if actualChecksum != expectedChecksum {
					cmd.Printf("✗ Checksum mismatch: %s\n", name)
					cmd.Printf("  Expected: %s\n", expectedChecksum)
					cmd.Printf("  Got:      %s\n", actualChecksum)
					filesCorrupt++
				} else {
					filesOk++
				}
			}

			if filesCorrupt > 0 {
				cmd.Printf("✗ Checksums: %d/%d files OK\n", filesOk, filesOk+filesCorrupt)
			} else {
				cmd.Printf("✓ Checksums: %d/%d files OK\n", filesOk, filesOk)
			}

			// Manifest completeness check
			untrackedFiles := 0
			walkErr := dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() && path != "manifest.json" {
					if _, exists := manifest[path]; !exists {
						untrackedFiles++
						cmd.Printf("✗ Untracked file: %s\n", path)
					}
				}
				return nil
			})
			if walkErr != nil {
				return fmt.Errorf("error checking for untracked files: %w", walkErr)
			}

			if filesCorrupt > 0 || untrackedFiles > 0 {
				cmd.Println("✗ Manifest: incomplete")
				return fmt.Errorf("%d files corrupted or untracked", filesCorrupt+untrackedFiles)
			}

			cmd.Println("✓ Manifest: complete")
			return nil
		},
	}
	cmd.Flags().String("source", "", "Original source to compare against")
	cmd.Flags().StringP("password", "p", "", "Password for decryption (for .stim files)")
	return cmd
}

func GetVerifyCmd() *cobra.Command {
	return verifyCmd
}

package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/spf13/cobra"
)

// repairCmd represents the repair command
var repairCmd = NewRepairCmd()

func init() {
	RootCmd.AddCommand(GetRepairCmd())
}
func NewRepairCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repair [archive]",
		Short: "Repair a corrupted archive.",
		Long:  `Repair a corrupted archive by re-downloading missing or corrupted files from the original source.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			archivePath := args[0]
			password, _ := cmd.Flags().GetString("password")
			source, _ := cmd.Flags().GetString("source")

			if source == "" {
				return fmt.Errorf("--source is required for repair")
			}

			cmd.Printf("Repairing %s...\n", archivePath)

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
					return fmt.Errorf("decryption failed: %w", err)
				}
				dn = t.RootFS
			} else {
				dn, err = datanode.FromTar(content)
				if err != nil {
					return fmt.Errorf("archive structure is corrupt: %w", err)
				}
			}

			manifestFile, err := dn.Open("manifest.json")
			if err != nil {
				return fmt.Errorf("could not open manifest: %w", err)
			}
			defer manifestFile.Close()

			manifestBytes, err := io.ReadAll(manifestFile)
			if err != nil {
				return fmt.Errorf("could not read manifest: %w", err)
			}

			var manifest map[string]string
			if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
				return fmt.Errorf("could not parse manifest: %w", err)
			}

			var corruptedFiles []string
			for name, expectedChecksum := range manifest {
				file, err := dn.Open(name)
				if err != nil {
					corruptedFiles = append(corruptedFiles, name)
					continue
				}
				defer file.Close()

				fileBytes, err := io.ReadAll(file)
				if err != nil {
					corruptedFiles = append(corruptedFiles, name)
					continue
				}

				hash := sha256.Sum256(fileBytes)
				actualChecksum := hex.EncodeToString(hash[:])

				if actualChecksum != expectedChecksum {
					corruptedFiles = append(corruptedFiles, name)
				}
			}

			if len(corruptedFiles) == 0 {
				cmd.Println("Archive is not corrupted.")
				return nil
			}

			cmd.Printf("Found %d corrupted files:\n", len(corruptedFiles))
			for _, file := range corruptedFiles {
				cmd.Printf(" - %s\n", file)
			}

			cmd.Println("Attempting to repair from source...")
			repairedCount := 0
			for _, file := range corruptedFiles {
				fileURL := source + "/" + file
				resp, err := http.Get(fileURL)
				if err != nil {
					cmd.Printf("  ✗ Could not download %s: %v\n", file, err)
					continue
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					cmd.Printf("  ✗ Could not download %s: status %s\n", file, resp.Status)
					continue
				}

				newContent, err := io.ReadAll(resp.Body)
				if err != nil {
					cmd.Printf("  ✗ Could not read downloaded content for %s: %v\n", file, err)
					continue
				}
				dn.ReplaceFile(file, newContent)
				repairedCount++
				cmd.Printf("  ✓ Repaired %s\n", file)
			}

			if repairedCount < len(corruptedFiles) {
				return fmt.Errorf("could not repair all corrupted files")
			}

			// Save the repaired archive
			repairedData, err := dn.ToTar()
			if err != nil {
				return fmt.Errorf("could not serialize repaired archive: %w", err)
			}

			if err := os.WriteFile(archivePath, repairedData, 0644); err != nil {
				return fmt.Errorf("could not write repaired archive: %w", err)
			}

			cmd.Println("Archive repaired successfully.")
			return nil
		},
	}
	cmd.Flags().String("source", "", "Original source to compare against")
	cmd.Flags().StringP("password", "p", "", "Password for decryption (for .stim files)")
	return cmd
}

func GetRepairCmd() *cobra.Command {
	return repairCmd
}

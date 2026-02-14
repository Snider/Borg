package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/diff"
	"github.com/spf13/cobra"
)

// NewDiffCmd creates a new diff command.
func NewDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <file1> <file2>",
		Short: "Compare two archives",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			file1Path := args[0]
			file2Path := args[1]

			// Read and decompress the first file
			file1Data, err := os.ReadFile(file1Path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", file1Path, err)
			}
			file1Data, err = compress.Decompress(file1Data)
			if err != nil {
				return fmt.Errorf("failed to decompress file %s: %w", file1Path, err)
			}
			dn1, err := datanode.FromTar(file1Data)
			if err != nil {
				return fmt.Errorf("failed to create datanode from %s: %w", file1Path, err)
			}

			// Read and decompress the second file
			file2Data, err := os.ReadFile(file2Path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", file2Path, err)
			}
			file2Data, err = compress.Decompress(file2Data)
			if err != nil {
				return fmt.Errorf("failed to decompress file %s: %w", file2Path, err)
			}
			dn2, err := datanode.FromTar(file2Data)
			if err != nil {
				return fmt.Errorf("failed to create datanode from %s: %w", file2Path, err)
			}

			// Compare the two datanodes
			differences, err := diff.Compare(dn1, dn2)
			if err != nil {
				return fmt.Errorf("failed to compare archives: %w", err)
			}

			// Print the results
			if len(differences.Added) == 0 && len(differences.Removed) == 0 && len(differences.Modified) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No differences found.")
				return nil
			}

			if len(differences.Added) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\nAdded (%d):\n", len(differences.Added))
				for _, file := range differences.Added {
					fmt.Fprintf(cmd.OutOrStdout(), "  + %s\n", file)
				}
			}

			if len(differences.Removed) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\nRemoved (%d):\n", len(differences.Removed))
				for _, file := range differences.Removed {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", file)
				}
			}

			if len(differences.Modified) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\nModified (%d):\n", len(differences.Modified))
				for _, file := range differences.Modified {
					fmt.Fprintf(cmd.OutOrStdout(), "  ~ %s\n", file)
				}
			}

			return nil
		},
	}
	return cmd
}

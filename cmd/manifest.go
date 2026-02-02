package cmd

import (
	"fmt"
	"os"

	"strings"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/manifest"
	"github.com/Snider/Enchantrix/pkg/trix"
	"github.com/spf13/cobra"
)

var manifestCmd = NewManifestCmd()

func NewManifestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "manifest [archive]",
		Short: "Generate a manifest from an archive.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			archivePath := args[0]
			data, err := os.ReadFile(archivePath)
			if err != nil {
				return fmt.Errorf("error reading archive: %w", err)
			}

			if strings.HasSuffix(archivePath, ".stim") {
				t, err := trix.Decode(data, "STIM", nil)
				if err == nil {
					if manifest, ok := t.Header["public_manifest"].(string); ok {
						fmt.Fprintln(cmd.OutOrStdout(), manifest)
						return nil
					}
				}
			}

			decompressedData, err := compress.Decompress(data)
			if err != nil {
				return fmt.Errorf("error decompressing archive: %w", err)
			}

			dn, err := datanode.FromTar(decompressedData)
			if err != nil {
				return fmt.Errorf("error reading datanode from archive: %w", err)
			}

			m, err := manifest.Generate(dn, archivePath, "unknown", false)
			if err != nil {
				return fmt.Errorf("error generating manifest: %w", err)
			}

			manifestData, err := m.ToJSON()
			if err != nil {
				return fmt.Errorf("error marshalling manifest: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(manifestData))

			return nil
		},
	}
}

func GetManifestCmd() *cobra.Command {
	return manifestCmd
}

func init() {
	RootCmd.AddCommand(GetManifestCmd())
}

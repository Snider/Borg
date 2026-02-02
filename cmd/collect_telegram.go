package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/telegram"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/Snider/Borg/pkg/ui"
	"github.com/spf13/cobra"
)

var collectTelegramCmd = NewCollectTelegramCmd()

func init() {
	GetCollectCmd().AddCommand(GetCollectTelegramCmd())
}

func GetCollectTelegramCmd() *cobra.Command {
	return collectTelegramCmd
}

func NewCollectTelegramCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telegram",
		Short: "Collect a Telegram export",
		Long:  `Collect a Telegram export and store it in a DataNode.`,
	}
	cmd.AddCommand(NewCollectTelegramImportCmd())
	return cmd
}

func NewCollectTelegramImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [path]",
		Short: "Import a Telegram export",
		Long:  `Import a Telegram export and store it in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			exportPath := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")
			password, _ := cmd.Flags().GetString("password")

			if format != "datanode" && format != "tim" && format != "trix" {
				return fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', or 'trix')", format)
			}

			prompter := ui.NewNonInteractivePrompter(ui.GetWebsiteQuote)
			prompter.Start()
			defer prompter.Stop()

			dn, err := telegram.Parse(exportPath)
			if err != nil {
				return fmt.Errorf("error parsing telegram export: %w", err)
			}

			if dn == nil {
				return fmt.Errorf("parsing telegram export resulted in an empty datanode")
			}

			var data []byte
			switch format {
			case "tim":
				t, err := tim.FromDataNode(dn)
				if err != nil {
					return fmt.Errorf("error creating tim: %w", err)
				}
				data, err = t.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing tim: %w", err)
				}
			case "trix":
				data, err = trix.ToTrix(dn, password)
				if err != nil {
					return fmt.Errorf("error serializing trix: %w", err)
				}
			default: // datanode
				data, err = dn.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing DataNode: %w", err)
				}
			}

			compressedData, err := compress.Compress(data, compression)
			if err != nil {
				return fmt.Errorf("error compressing data: %w", err)
			}

			if outputFile == "" {
				outputFile = "telegram." + format
				if compression != "none" {
					outputFile += "." + compression
				}
			}

			err = os.WriteFile(outputFile, compressedData, 0644)
			if err != nil {
				return fmt.Errorf("error writing telegram export to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Telegram export saved to", outputFile)
			return nil
		},
	}
	cmd.Flags().String("output", "", "Output file for the DataNode")
	cmd.Flags().String("format", "datanode", "Output format (datanode, tim, or trix)")
	cmd.Flags().String("compression", "none", "Compression format (none, gz, or xz)")
	cmd.Flags().String("password", "", "Password for encryption")
	return cmd
}

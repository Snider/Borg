package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/Snider/Borg/pkg/pdf"
	"github.com/spf13/cobra"
)

// pdfMetadataCmd represents the pdf metadata command
var pdfMetadataCmd = NewPdfMetadataCmd()

func init() {
	GetPdfCmd().AddCommand(GetPdfMetadataCmd())
}

func NewPdfMetadataCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "metadata [file]",
		Short: "Extract metadata from a PDF file.",
		Long:  `Extract metadata from a PDF file and print it as JSON.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			metadata, err := pdf.ExtractMetadata(filePath)
			if err != nil {
				return fmt.Errorf("error extracting metadata: %w", err)
			}
			jsonMetadata, err := json.MarshalIndent(metadata, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling metadata to JSON: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(jsonMetadata))
			return nil
		},
	}
}

func GetPdfMetadataCmd() *cobra.Command {
	return pdfMetadataCmd
}

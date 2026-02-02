package cmd

import (
	"github.com/spf13/cobra"
)

// pdfCmd represents the pdf command
var pdfCmd = NewPdfCmd()

func init() {
	RootCmd.AddCommand(GetPdfCmd())
}

func NewPdfCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pdf",
		Short: "Perform PDF operations.",
		Long:  `A command for performing various PDF operations.`,
	}
}

func GetPdfCmd() *cobra.Command {
	return pdfCmd
}

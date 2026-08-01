package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	trixsdk "github.com/Snider/Enchantrix/pkg/trix"
	"github.com/spf13/cobra"
)

var inspectCmd = NewInspectCmd()

func NewInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [file]",
		Short: "Inspect metadata of a .trix or .stim file without decrypting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputFile := args[0]
			jsonOutput, _ := cmd.Flags().GetBool("json")

			data, err := os.ReadFile(inputFile)
			if err != nil {
				return err
			}

			if len(data) < 4 {
				return fmt.Errorf("file too small to be a valid container")
			}

			magic := string(data[:4])
			var t *trixsdk.Trix

			switch magic {
			case "STIM":
				t, err = trixsdk.Decode(data, "STIM", nil)
				if err != nil {
					return fmt.Errorf("failed to decode STIM: %w", err)
				}
			case "TRIX":
				t, err = trixsdk.Decode(data, "TRIX", nil)
				if err != nil {
					return fmt.Errorf("failed to decode TRIX: %w", err)
				}
			default:
				return fmt.Errorf("unknown file format (magic: %q)", magic)
			}

			if jsonOutput {
				info := map[string]interface{}{
					"file":         inputFile,
					"magic":        magic,
					"header":       t.Header,
					"payload_size": len(t.Payload),
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}

			// Human-readable output
			fmt.Fprintf(cmd.OutOrStdout(), "File: %s\n", inputFile)
			fmt.Fprintf(cmd.OutOrStdout(), "Format: %s\n", magic)
			fmt.Fprintf(cmd.OutOrStdout(), "Payload Size: %d bytes\n", len(t.Payload))
			fmt.Fprintf(cmd.OutOrStdout(), "Header:\n")

			for k, v := range t.Header {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: %v\n", k, v)
			}

			// Show encryption info
			if algo, ok := t.Header["encryption_algorithm"]; ok {
				fmt.Fprintf(cmd.OutOrStdout(), "\nEncryption: %v\n", algo)
			}
			if _, ok := t.Header["tim"]; ok {
				fmt.Fprintf(cmd.OutOrStdout(), "Type: Terminal Isolation Matrix\n")
			}
			if v, ok := t.Header["version"]; ok {
				fmt.Fprintf(cmd.OutOrStdout(), "Version: %v\n", v)
			}

			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	return cmd
}

func GetInspectCmd() *cobra.Command {
	return inspectCmd
}

func init() {
	RootCmd.AddCommand(GetInspectCmd())
}

// isStimFile checks if a file is a .stim file by extension or magic number.
func isStimFile(path string) bool {
	if strings.HasSuffix(path, ".stim") {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	magic := make([]byte, 4)
	if _, err := f.Read(magic); err != nil {
		return false
	}
	return string(magic) == "STIM"
}

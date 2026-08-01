package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	trixsdk "github.com/Snider/Enchantrix/pkg/trix"
	"github.com/spf13/cobra"
)

var decodeCmd = NewDecodeCmd()

func NewDecodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decode [file]",
		Short: "Decode a .trix, .tim, or .stim file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputFile := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			password, _ := cmd.Flags().GetString("password")
			inIsolation, _ := cmd.Flags().GetBool("i-am-in-isolation")

			data, err := os.ReadFile(inputFile)
			if err != nil {
				return err
			}

			// Check if it's a .stim file (encrypted TIM)
			if strings.HasSuffix(inputFile, ".stim") || (len(data) >= 4 && string(data[:4]) == "STIM") {
				if password == "" {
					return fmt.Errorf("password required for .stim files")
				}
				if !inIsolation {
					return fmt.Errorf("this is an encrypted Terminal Isolation Matrix, use the --i-am-in-isolation flag to decode it")
				}
				m, err := tim.FromSigil(data, password)
				if err != nil {
					return err
				}
				tarball, err := m.ToTar()
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Decoded encrypted TIM to %s\n", outputFile)
				return os.WriteFile(outputFile, tarball, 0644)
			}

			// Try TRIX format
			t, err := trixsdk.Decode(data, "TRIX", nil)
			if err != nil {
				return err
			}

			if _, ok := t.Header["tim"]; ok && !inIsolation {
				return fmt.Errorf("this is a Terminal Isolation Matrix, use the --i-am-in-isolation flag to decode it")
			}

			dn, err := trix.FromTrix(data, password)
			if err != nil {
				return err
			}

			tarball, err := dn.ToTar()
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Decoded to %s\n", outputFile)
			return os.WriteFile(outputFile, tarball, 0644)
		},
	}
	cmd.Flags().String("output", "decoded.dat", "Output file for the decoded data")
	cmd.Flags().String("password", "", "Password for decryption")
	cmd.Flags().Bool("i-am-in-isolation", false, "Required to decode a Terminal Isolation Matrix")
	return cmd
}

func GetDecodeCmd() *cobra.Command {
	return decodeCmd
}

func init() {
	RootCmd.AddCommand(GetDecodeCmd())
}

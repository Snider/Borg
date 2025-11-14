package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/trix"
	trixsdk "github.com/Snider/Enchantrix/pkg/trix"
	"github.com/spf13/cobra"
)

var decodeCmd = NewDecodeCmd()

func NewDecodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decode [file]",
		Short: "Decode a .trix or .tim file",
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

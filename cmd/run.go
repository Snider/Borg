package cmd

import (
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/tim"
	"github.com/spf13/cobra"
)

var runPassword string

var runCmd = NewRunCmd()

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [tim file]",
		Short: "Run a Terminal Isolation Matrix.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			// Check if encrypted by extension or magic number
			if isEncryptedTIM(filePath) {
				password, _ := cmd.Flags().GetString("password")
				if password == "" {
					return tim.ErrPasswordRequired
				}
				return tim.RunEncrypted(filePath, password)
			}

			return tim.Run(filePath)
		},
	}
	cmd.Flags().StringVarP(&runPassword, "password", "p", "", "Decryption password for encrypted TIMs (.stim)")
	return cmd
}

// isEncryptedTIM checks if a file is an encrypted TIM by extension or magic number.
func isEncryptedTIM(path string) bool {
	if strings.HasSuffix(path, ".stim") {
		return true
	}
	// Check magic number
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

func GetRunCmd() *cobra.Command {
	return runCmd
}

func init() {
	RootCmd.AddCommand(GetRunCmd())
}

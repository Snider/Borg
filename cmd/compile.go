package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/tim"
	"github.com/spf13/cobra"
)

var borgfile string
var output string
var encryptPassword string

var compileCmd = NewCompileCmd()

func NewCompileCmd() *cobra.Command {
	compileCmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile a Borgfile into a Terminal Isolation Matrix.",
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := os.ReadFile(borgfile)
			if err != nil {
				return err
			}

			m, err := tim.New()
			if err != nil {
				return err
			}

			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				parts := strings.Fields(line)
				if len(parts) == 0 {
					continue
				}
				switch parts[0] {
				case "ADD":
					if len(parts) != 3 {
						return fmt.Errorf("invalid ADD instruction: %s", line)
					}
					src := parts[1]
					dest := parts[2]
					data, err := os.ReadFile(src)
					if err != nil {
						return err
					}
					m.RootFS.AddData(strings.TrimPrefix(dest, "/"), data)
				default:
					return fmt.Errorf("unknown instruction: %s", parts[0])
				}
			}

			// If encryption is requested, output as .stim
			if encryptPassword != "" {
				stimData, err := m.ToSigil(encryptPassword)
				if err != nil {
					return err
				}
				outputPath := output
				if !strings.HasSuffix(outputPath, ".stim") {
					outputPath = strings.TrimSuffix(outputPath, ".tim") + ".stim"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Compiled encrypted TIM to %s\n", outputPath)
				return os.WriteFile(outputPath, stimData, 0644)
			}

			// Original unencrypted output
			tarball, err := m.ToTar()
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Compiled TIM to %s\n", output)
			return os.WriteFile(output, tarball, 0644)
		},
	}
	compileCmd.Flags().StringVarP(&borgfile, "file", "f", "Borgfile", "Path to the Borgfile.")
	compileCmd.Flags().StringVarP(&output, "output", "o", "a.tim", "Path to the output tim file.")
	compileCmd.Flags().StringVarP(&encryptPassword, "encrypt", "e", "", "Encrypt with ChaCha20-Poly1305 using this password (outputs .stim)")
	return compileCmd
}

func GetCompileCmd() *cobra.Command {
	return compileCmd
}

func init() {
	RootCmd.AddCommand(GetCompileCmd())
}

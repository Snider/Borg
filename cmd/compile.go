package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/matrix"
	"github.com/spf13/cobra"
)

var borgfile string
var output string

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile a Borgfile into a Terminal Isolation Matrix.",
	RunE: func(cmd *cobra.Command, args []string) error {
		content, err := os.ReadFile(borgfile)
		if err != nil {
			return err
		}

		m, err := matrix.New()
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
				m.RootFS.AddData(dest, data)
			default:
				return fmt.Errorf("unknown instruction: %s", parts[0])
			}
		}

		tarball, err := m.ToTar()
		if err != nil {
			return err
		}

		return os.WriteFile(output, tarball, 0644)
	},
}

func init() {
	RootCmd.AddCommand(compileCmd)
	compileCmd.Flags().StringVarP(&borgfile, "file", "f", "Borgfile", "Path to the Borgfile.")
	compileCmd.Flags().StringVarP(&output, "output", "o", "a.matrix", "Path to the output matrix file.")
}

package cmd

import (
	"fmt"
	"io"
	"os"

	"borg-data-collector/pkg/trix"

	"github.com/spf13/cobra"
)

// catCmd represents the cat command
var catCmd = &cobra.Command{
	Use:   "cat [cube-file] [file-to-extract]",
	Short: "Extract a file from a Trix cube",
	Long: `Extract a file from a Trix cube and print its content to standard output.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cubeFile := args[0]
		fileToExtract := args[1]

		reader, file, err := trix.Extract(cubeFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		for {
			hdr, err := reader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println(err)
				return
			}

			if hdr.Name == fileToExtract {
				if _, err := io.Copy(os.Stdout, reader); err != nil {
					fmt.Println(err)
					return
				}
				return
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(catCmd)
}

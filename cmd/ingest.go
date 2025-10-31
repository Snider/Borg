package cmd

import (
	"fmt"
	"os"

	"borg-data-collector/pkg/borg"
	"borg-data-collector/pkg/trix"

	"github.com/spf13/cobra"
)

// ingestCmd represents the ingest command
var ingestCmd = &cobra.Command{
	Use:   "ingest [cube-file] [file-to-add]",
	Short: "Add a file to a Trix cube",
	Long: `Add a file to a Trix cube. If the cube file does not exist, it will be created.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cubeFile := args[0]
		fileToAdd := args[1]

		var cube *trix.Cube
		var err error

		if _, err := os.Stat(cubeFile); os.IsNotExist(err) {
			cube, err = trix.NewCube(cubeFile)
		} else {
			cube, err = trix.AppendToCube(cubeFile)
		}

		if err != nil {
			fmt.Println(err)
			return
		}
		defer cube.Close()

		content, err := os.ReadFile(fileToAdd)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = cube.AddFile(fileToAdd, content)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(borg.GetRandomCodeShortMessage())
	},
}

func init() {
	rootCmd.AddCommand(ingestCmd)
}

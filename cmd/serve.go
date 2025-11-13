package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tarfs"

	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve [file]",
	Short: "Serve a packaged PWA file",
	Long:  `Serves the contents of a packaged PWA file using a static file server.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dataFile := args[0]
		port, _ := cmd.Flags().GetString("port")

		rawData, err := os.ReadFile(dataFile)
		if err != nil {
			return fmt.Errorf("Error reading data file: %w", err)
		}

		data, err := compress.Decompress(rawData)
		if err != nil {
			return fmt.Errorf("Error decompressing data: %w", err)
		}

		var fs http.FileSystem
		if strings.HasSuffix(dataFile, ".matrix") {
			fs, err = tarfs.New(data)
			if err != nil {
				return fmt.Errorf("Error creating TarFS from matrix tarball: %w", err)
			}
		} else {
			dn, err := datanode.FromTar(data)
			if err != nil {
				return fmt.Errorf("Error creating DataNode from tarball: %w", err)
			}
			fs = http.FS(dn)
		}

		http.Handle("/", http.FileServer(fs))

		fmt.Printf("Serving PWA on http://localhost:%s\n", port)
		err = http.ListenAndServe(":"+port, nil)
		if err != nil {
			return fmt.Errorf("Error starting server: %w", err)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
	serveCmd.PersistentFlags().String("port", "8080", "Port to serve the PWA on")
}

package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Snider/Borg/pkg/datanode"

	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve [file]",
	Short: "Serve a packaged PWA file",
	Long:  `Serves the contents of a packaged PWA file using a static file server.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pwaFile := args[0]
		port, _ := cmd.Flags().GetString("port")

		pwaData, err := os.ReadFile(pwaFile)
		if err != nil {
			fmt.Printf("Error reading PWA file: %v\n", err)
			return
		}

		dn, err := datanode.FromTar(pwaData)
		if err != nil {
			fmt.Printf("Error creating DataNode from tarball: %v\n", err)
			return
		}

		http.Handle("/", http.FileServer(http.FS(dn)))

		fmt.Printf("Serving PWA on http://localhost:%s\n", port)
		err = http.ListenAndServe(":"+port, nil)
		if err != nil {
			fmt.Printf("Error starting server: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.PersistentFlags().String("port", "8080", "Port to serve the PWA on")
}

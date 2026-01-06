// dapp-fm CLI provides headless media player functionality
// For native desktop app with WebView, use dapp-fm-app instead
package main

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/player"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dapp-fm",
		Short: "dapp.fm - Decentralized Music Player CLI",
		Long: `dapp-fm is the CLI version of the dapp.fm player.

For the native desktop app with WebView, use dapp-fm-app instead.
This CLI provides HTTP server mode for automation and fallback scenarios.`,
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server for the media player",
		Long: `Starts an HTTP server serving the media player interface.
This is the slower TCP path - for memory-speed decryption, use dapp-fm-app.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetString("port")
			openBrowser, _ := cmd.Flags().GetBool("open")

			p := player.NewPlayer()

			addr := ":" + port
			if openBrowser {
				fmt.Printf("Opening browser at http://localhost%s\n", addr)
				// Would need browser opener here
			}

			return p.Serve(addr)
		},
	}

	serveCmd.Flags().StringP("port", "p", "8080", "Port to serve on")
	serveCmd.Flags().Bool("open", false, "Open browser automatically")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("dapp-fm v1.0.0")
			fmt.Println("Decentralized Music Distribution")
			fmt.Println("https://dapp.fm")
		},
	}

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

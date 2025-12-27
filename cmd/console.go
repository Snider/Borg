package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Snider/Borg/pkg/console"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/spf13/cobra"
)

var consoleCmd = NewConsoleCmd()

// NewConsoleCmd creates the console parent command.
func NewConsoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "console",
		Short: "Manage encrypted PWA console demos",
		Long: `The Borg Console packages and serves encrypted PWA demos.

Build a console STIM:
  borg console build -p "password" -o console.stim

Serve with unlock page:
  borg console serve console.stim --open

Serve pre-unlocked:
  borg console serve console.stim -p "password" --open`,
	}

	cmd.AddCommand(NewConsoleBuildCmd())
	cmd.AddCommand(NewConsoleServeCmd())

	return cmd
}

// NewConsoleBuildCmd creates the build subcommand.
func NewConsoleBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a console STIM from demo files",
		Long: `Packages HTML demo files into an encrypted STIM container.

By default, looks for files in js/borg-stmf/ directory.
Required files: index.html, support-reply.html, stmf.wasm, wasm_exec.js`,
		RunE: func(cmd *cobra.Command, args []string) error {
			password, _ := cmd.Flags().GetString("password")
			output, _ := cmd.Flags().GetString("output")
			sourceDir, _ := cmd.Flags().GetString("source")

			if password == "" {
				return fmt.Errorf("password is required")
			}

			// Create new TIM
			m, err := tim.New()
			if err != nil {
				return fmt.Errorf("creating TIM: %w", err)
			}

			// Required demo files
			files := []string{
				"index.html",
				"support-reply.html",
				"stmf.wasm",
				"wasm_exec.js",
			}

			// Add each file to the TIM
			for _, f := range files {
				path := filepath.Join(sourceDir, f)
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("reading %s: %w", f, err)
				}
				m.RootFS.AddData(f, data)
				fmt.Printf("  + %s (%d bytes)\n", f, len(data))
			}

			// Encrypt to STIM
			stim, err := m.ToSigil(password)
			if err != nil {
				return fmt.Errorf("encrypting STIM: %w", err)
			}

			// Write output
			if err := os.WriteFile(output, stim, 0644); err != nil {
				return fmt.Errorf("writing output: %w", err)
			}

			fmt.Printf("\nBuilt: %s (%d bytes)\n", output, len(stim))
			fmt.Println("Encrypted with ChaCha20-Poly1305")

			return nil
		},
	}

	cmd.Flags().StringP("password", "p", "", "Encryption password (required)")
	cmd.Flags().StringP("output", "o", "console.stim", "Output file")
	cmd.Flags().StringP("source", "s", "js/borg-stmf", "Source directory")
	cmd.MarkFlagRequired("password")

	return cmd
}

// NewConsoleServeCmd creates the serve subcommand.
func NewConsoleServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve [stim-file]",
		Short: "Serve an encrypted console STIM",
		Long: `Starts an HTTP server to serve encrypted STIM content.

Without a password, shows a dark-themed unlock page.
With a password, decrypts immediately and serves content.

Examples:
  borg console serve demos.stim --open
  borg console serve demos.stim -p "password" --port 3000`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stimPath := args[0]
			password, _ := cmd.Flags().GetString("password")
			port, _ := cmd.Flags().GetString("port")
			openBrowser, _ := cmd.Flags().GetBool("open")

			// Create server
			server, err := console.NewServer(stimPath, password, port)
			if err != nil {
				return err
			}

			// Print status
			fmt.Printf("Borg Console serving at %s\n", server.URL())
			if password != "" {
				fmt.Println("Status: Unlocked (password provided)")
			} else {
				fmt.Println("Status: Locked (unlock page active)")
			}
			fmt.Println()

			// Open browser if requested
			if openBrowser {
				if err := console.OpenBrowser(server.URL()); err != nil {
					fmt.Printf("Warning: could not open browser: %v\n", err)
				}
			}

			// Start serving
			return server.Start()
		},
	}

	cmd.Flags().StringP("password", "p", "", "Decryption password (skip unlock page)")
	cmd.Flags().String("port", "8080", "Port to serve on")
	cmd.Flags().Bool("open", false, "Auto-open browser")

	return cmd
}

func init() {
	RootCmd.AddCommand(consoleCmd)
}

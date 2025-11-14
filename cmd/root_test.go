package cmd

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// executeCommand is a helper function to execute a cobra command and return the output.
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	_, output, err := executeCommandC(root, args...)
	return output, err
}

// executeCommandC is a helper function to execute a cobra command and return the output.
func executeCommandC(root *cobra.Command, args ...string) (*cobra.Command, string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	c, err := root.ExecuteC()

	return c, buf.String(), err
}

func TestExecute_Good(t *testing.T) {
	// This is a basic test to ensure the command runs without panicking.
	err := Execute(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootCmd_Good(t *testing.T) {
	t.Run("No args", func(t *testing.T) {
		_, err := executeCommand(RootCmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Help flag", func(t *testing.T) {
		// We need to reset the command's state before each run.
		RootCmd.ResetFlags()
		RootCmd.ResetCommands()
		initAllCommands()

		output, err := executeCommand(RootCmd, "--help")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "Usage:") {
			t.Errorf("expected help output to contain 'Usage:', but it did not")
		}
	})
}

func TestRootCmd_Bad(t *testing.T) {
	t.Run("Unknown command", func(t *testing.T) {
		// We need to reset the command's state before each run.
		RootCmd.ResetFlags()
		RootCmd.ResetCommands()
		initAllCommands()

		_, err := executeCommand(RootCmd, "unknown-command")
		if err == nil {
			t.Fatal("expected an error for an unknown command, but got none")
		}
	})
}

// initAllCommands re-initializes all commands for testing.
func initAllCommands() {
	RootCmd.AddCommand(GetAllCmd())
	RootCmd.AddCommand(GetCollectCmd())
	RootCmd.AddCommand(GetCompileCmd())
	RootCmd.AddCommand(GetRunCmd())
	RootCmd.AddCommand(GetServeCmd())
}

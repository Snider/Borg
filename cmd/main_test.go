package cmd

import (
	"bytes"

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

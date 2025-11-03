package cmd

import (
	"strings"
	"testing"
)

func TestCollectPWACmd_NoURI(t *testing.T) {
	rootCmd := NewRootCmd()
	collectCmd := NewCollectCmd()
	collectPWACmd := NewCollectPWACmd()
	collectCmd.AddCommand(&collectPWACmd.Command)
	rootCmd.AddCommand(collectCmd)
	_, err := executeCommand(rootCmd, "collect", "pwa")
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}
	if !strings.Contains(err.Error(), "uri is required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
func Test_NewCollectPWACmd(t *testing.T) {
	if NewCollectPWACmd() == nil {
		t.Errorf("NewCollectPWACmd is nil")
	}
}

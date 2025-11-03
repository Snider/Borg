package cmd

import (
	"strings"
	"testing"
)

func TestCollectWebsiteCmd_NoArgs(t *testing.T) {
	rootCmd := NewRootCmd()
	collectCmd := NewCollectCmd()
	collectWebsiteCmd := NewCollectWebsiteCmd()
	collectCmd.AddCommand(collectWebsiteCmd)
	rootCmd.AddCommand(collectCmd)
	_, err := executeCommand(rootCmd, "collect", "website")
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
func Test_NewCollectWebsiteCmd(t *testing.T) {
	if NewCollectWebsiteCmd() == nil {
		t.Errorf("NewCollectWebsiteCmd is nil")
	}
}

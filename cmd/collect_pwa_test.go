package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/pwa"
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

func TestCollectPWA_Good(t *testing.T) {
	mockClient := &pwa.MockPWAClient{
		ManifestURL: "https://example.com/manifest.json",
		DN:          datanode.New(),
		Err:         nil,
	}

	_, err := CollectPWA(mockClient, "https://example.com", "/dev/null", "datanode", "none")
	if err != nil {
		t.Fatalf("CollectPWA failed: %v", err)
	}
}

func TestCollectPWA_Bad(t *testing.T) {
	mockClient := &pwa.MockPWAClient{
		ManifestURL: "",
		DN:          nil,
		Err:         fmt.Errorf("pwa error"),
	}

	_, err := CollectPWA(mockClient, "https://example.com", "/dev/null", "datanode", "none")
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}
}

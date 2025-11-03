package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/website"
	"github.com/schollz/progressbar/v3"
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

func TestCollectWebsiteCmd_Good(t *testing.T) {
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
		return datanode.New(), nil
	}
	defer func() {
		website.DownloadAndPackageWebsite = oldDownloadAndPackageWebsite
	}()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(collectCmd)

	out := filepath.Join(t.TempDir(), "out")
	_, err := executeCommand(rootCmd, "collect", "website", "https://example.com", "--output", out)
	if err != nil {
		t.Fatalf("collect website command failed: %v", err)
	}
}

func TestCollectWebsiteCmd_Bad(t *testing.T) {
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
		return nil, fmt.Errorf("website error")
	}
	defer func() {
		website.DownloadAndPackageWebsite = oldDownloadAndPackageWebsite
	}()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(collectCmd)

	out := filepath.Join(t.TempDir(), "out")
	_, err := executeCommand(rootCmd, "collect", "website", "https://example.com", "--output", out)
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}
}

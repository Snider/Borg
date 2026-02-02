package cmd

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/website"
	"github.com/schollz/progressbar/v3"
)

func TestCollectWebsiteCmd_Good(t *testing.T) {
	// Mock the website downloader
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
		return datanode.New(), nil
	}
	defer func() {
		website.DownloadAndPackageWebsite = oldDownloadAndPackageWebsite
	}()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())

	// Execute command
	out := filepath.Join(t.TempDir(), "out")
	_, err := executeCommand(rootCmd, "collect", "website", "https://example.com", "--output", out)
	if err != nil {
		t.Fatalf("collect website command failed: %v", err)
	}
}

type mockDownloader struct {
	err error
}

func (m *mockDownloader) Download(startURL string) (*datanode.DataNode, error) {
	return nil, m.err
}

func (m *mockDownloader) SetProgressBar(bar *progressbar.ProgressBar) {
	// do nothing
}

func TestCollectWebsiteCmd_Bad(t *testing.T) {
	oldNewDownloader := website.NewDownloaderWithClient
	website.NewDownloaderWithClient = func(maxDepth int, client *http.Client) website.Downloader {
		return &mockDownloader{err: fmt.Errorf("website error")}
	}
	t.Cleanup(func() {
		website.NewDownloaderWithClient = oldNewDownloader
	})

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())

	// Execute command
	out := filepath.Join(t.TempDir(), "out")
	_, err := executeCommand(rootCmd, "collect", "website", "https://example.com", "--output", out)
	if err == nil {
		t.Fatal("expected an error, but got none")
	}
}

func TestCollectWebsiteCmd_Ugly(t *testing.T) {
	t.Run("No arguments", func(t *testing.T) {
		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetCollectCmd())
		_, err := executeCommand(rootCmd, "collect", "website")
		if err == nil {
			t.Fatal("expected an error for no arguments, but got none")
		}
		if !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

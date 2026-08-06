package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/website"
	"github.com/schollz/progressbar/v3"
)

func TestCollectWebsiteCmd_Good(t *testing.T) {
	// Mock the website downloader
	oldDownloadAndPackageWebsiteWithClient := website.DownloadAndPackageWebsiteWithClient
	website.DownloadAndPackageWebsiteWithClient = func(startURL string, maxDepth int, bar *progressbar.ProgressBar, client *http.Client) (*datanode.DataNode, error) {
		return datanode.New(), nil
	}
	defer func() {
		website.DownloadAndPackageWebsiteWithClient = oldDownloadAndPackageWebsiteWithClient
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

func TestCollectWebsiteCmd_Bad(t *testing.T) {
	// Mock the website downloader to return an error
	oldDownloadAndPackageWebsiteWithClient := website.DownloadAndPackageWebsiteWithClient
	website.DownloadAndPackageWebsiteWithClient = func(startURL string, maxDepth int, bar *progressbar.ProgressBar, client *http.Client) (*datanode.DataNode, error) {
		return nil, fmt.Errorf("website error")
	}
	defer func() {
		website.DownloadAndPackageWebsiteWithClient = oldDownloadAndPackageWebsiteWithClient
	}()

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

	t.Run("Invalid bandwidth", func(t *testing.T) {
		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetCollectCmd())
		_, err := executeCommand(rootCmd, "collect", "website", "https://example.com", "--bandwidth", "1Gbps")
		if err == nil {
			t.Fatal("expected an error for invalid bandwidth, but got none")
		}
		if !strings.Contains(err.Error(), "invalid bandwidth") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestCollectWebsiteCmd_Bandwidth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 1024*1024)) // 1MB
	}))
	defer server.Close()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())

	// Create a temporary directory for the output file
	outDir := t.TempDir()
	out := filepath.Join(outDir, "out")

	// Execute command with a bandwidth limit
	start := time.Now()
	_, err := executeCommand(rootCmd, "collect", "website", server.URL, "--output", out, "--bandwidth", "500KB/s")
	if err != nil {
		t.Fatalf("collect website command failed: %v", err)
	}
	elapsed := time.Since(start)

	// Check if the download took at least 2 seconds
	if elapsed < 2*time.Second {
		t.Errorf("expected download to take at least 2 seconds, but it took %s", elapsed)
	}
}

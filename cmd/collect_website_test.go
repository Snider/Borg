package cmd

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	borghttp "github.com/Snider/Borg/pkg/http"
	"github.com/Snider/Borg/pkg/website"
	"github.com/schollz/progressbar/v3"
)

func TestCollectWebsiteCmd_Good(t *testing.T) {
	// Mock the website downloader
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, bar *progressbar.ProgressBar, client *http.Client) (*datanode.DataNode, error) {
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

func TestCollectWebsiteCmd_Bad(t *testing.T) {
	// Mock the website downloader to return an error
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, bar *progressbar.ProgressBar, client *http.Client) (*datanode.DataNode, error) {
		return nil, fmt.Errorf("website error")
	}
	defer func() {
		website.DownloadAndPackageWebsite = oldDownloadAndPackageWebsite
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

func TestCollectWebsiteCmd_RateLimit(t *testing.T) {
	var capturedClient *http.Client
	// Mock the website downloader
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, bar *progressbar.ProgressBar, client *http.Client) (*datanode.DataNode, error) {
		capturedClient = client
		return datanode.New(), nil
	}
	defer func() {
		website.DownloadAndPackageWebsite = oldDownloadAndPackageWebsite
	}()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())

	// Execute command
	out := filepath.Join(t.TempDir(), "out")
	_, err := executeCommand(rootCmd, "collect", "website", "https://example.com", "--output", out, "--rate-limit", "10/s", "--burst", "5")
	if err != nil {
		t.Fatalf("collect website command failed: %v", err)
	}

	if capturedClient == nil {
		t.Fatal("http client was not passed to the downloader")
	}

	if _, ok := capturedClient.Transport.(*borghttp.RateLimitingRoundTripper); !ok {
		t.Errorf("expected a rate limiting transport, but got %T", capturedClient.Transport)
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

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

func TestCollectWebsiteCmd_Good(t *testing.T) {
	// Mock the website downloader
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, useSitemap, sitemapOnly bool, sitemapURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
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

func TestCollectWebsiteCmd_Sitemap_Good(t *testing.T) {
	var capturedUseSitemap, capturedSitemapOnly bool
	var capturedSitemapURL string

	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, useSitemap, sitemapOnly bool, sitemapURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
		capturedUseSitemap = useSitemap
		capturedSitemapOnly = sitemapOnly
		capturedSitemapURL = sitemapURL
		return datanode.New(), nil
	}
	defer func() {
		website.DownloadAndPackageWebsite = oldDownloadAndPackageWebsite
	}()

	testCases := []struct {
		name                string
		args                []string
		expectedUseSitemap  bool
		expectedSitemapOnly bool
		expectedSitemapURL  string
	}{
		{
			name:                "use-sitemap flag",
			args:                []string{"https://example.com", "--use-sitemap"},
			expectedUseSitemap:  true,
			expectedSitemapOnly: false,
			expectedSitemapURL:  "",
		},
		{
			name:                "sitemap-only flag",
			args:                []string{"https://example.com", "--sitemap-only"},
			expectedUseSitemap:  false,
			expectedSitemapOnly: true,
			expectedSitemapURL:  "",
		},
		{
			name:                "sitemap flag",
			args:                []string{"https://example.com", "--sitemap", "https://example.com/sitemap.xml"},
			expectedUseSitemap:  false,
			expectedSitemapOnly: false,
			expectedSitemapURL:  "https://example.com/sitemap.xml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd := NewCollectWebsiteCmd()
			_, err := executeCommand(rootCmd, tc.args...)
			if err != nil {
				t.Fatalf("command execution failed: %v", err)
			}

			if capturedUseSitemap != tc.expectedUseSitemap {
				t.Errorf("expected useSitemap to be %v, but got %v", tc.expectedUseSitemap, capturedUseSitemap)
			}
			if capturedSitemapOnly != tc.expectedSitemapOnly {
				t.Errorf("expected sitemapOnly to be %v, but got %v", tc.expectedSitemapOnly, capturedSitemapOnly)
			}
			if capturedSitemapURL != tc.expectedSitemapURL {
				t.Errorf("expected sitemapURL to be %q, but got %q", tc.expectedSitemapURL, capturedSitemapURL)
			}
		})
	}
}

func TestCollectWebsiteCmd_Bad(t *testing.T) {
	// Mock the website downloader to return an error
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, useSitemap, sitemapOnly bool, sitemapURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
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

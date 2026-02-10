package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/hooks"
	"github.com/Snider/Borg/pkg/website"
	"github.com/schollz/progressbar/v3"
	"github.com/stretchr/testify/require"
)

func TestCollectWebsiteCmd_Good(t *testing.T) {
	// Mock the website downloader
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, bar *progressbar.ProgressBar, hookRunner *hooks.HookRunner) (*datanode.DataNode, error) {
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

func TestCollectWebsiteCmd_Hooks(t *testing.T) {
	// 1. Setup temp directory for test artifacts
	tmpDir := t.TempDir()

	// 2. Create the hook script
	scriptContent := "#!/bin/sh\ncat > " + filepath.Join(tmpDir, "hook.output")
	scriptPath := filepath.Join(tmpDir, "testhook.sh")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// 3. Create the hooks YAML config
	hooksYAML := `
hooks:
  on_collection_complete:
    - run: "` + scriptPath + `"
`
	configPath := filepath.Join(tmpDir, ".borg-hooks.yaml")
	err = os.WriteFile(configPath, []byte(hooksYAML), 0644)
	require.NoError(t, err)

	// 4. Mock the website downloader
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, bar *progressbar.ProgressBar, hookRunner *hooks.HookRunner) (*datanode.DataNode, error) {
		dn := datanode.New()
		// Manually trigger the hook that the real function would trigger
		err := hookRunner.Trigger(hooks.Event{
			Event: hooks.OnCollectionComplete,
		})
		require.NoError(t, err) // Use require in the mock to fail fast if the trigger fails
		return dn, nil
	}
	defer func() {
		website.DownloadAndPackageWebsite = oldDownloadAndPackageWebsite
	}()

	// 5. Execute the command
	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())
	out := filepath.Join(tmpDir, "out")
	_, err = executeCommand(rootCmd, "collect", "website", "https://example.com", "--output", out, "--hooks", configPath)
	require.NoError(t, err)

	// 6. Assert results
	hookOutputFile := filepath.Join(tmpDir, "hook.output")
	content, err := os.ReadFile(hookOutputFile)
	require.NoError(t, err, "Hook output file should have been created")

	var receivedEvent hooks.Event
	err = json.Unmarshal(content, &receivedEvent)
	require.NoError(t, err, "Failed to unmarshal hook output")

	require.Equal(t, hooks.OnCollectionComplete, receivedEvent.Event)
}

func TestCollectWebsiteCmd_Bad(t *testing.T) {
	// Mock the website downloader to return an error
	oldDownloadAndPackageWebsite := website.DownloadAndPackageWebsite
	website.DownloadAndPackageWebsite = func(startURL string, maxDepth int, bar *progressbar.ProgressBar, hookRunner *hooks.HookRunner) (*datanode.DataNode, error) {
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

package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (string, func()) {
	t.Helper()

	// Create a temporary directory for test artifacts
	tmpDir, err := os.MkdirTemp("", "hooks-test-")
	require.NoError(t, err)

	// Create test hook script
	scriptContent := "#!/bin/sh\ncat > " + filepath.Join(tmpDir, "testhook.output")
	scriptPath := filepath.Join(tmpDir, "testhook.sh")
	err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// Create test hooks config
	hooksYAML := `
hooks:
  on_file_collected:
    - pattern: "*.pdf"
      run: "` + scriptPath + `"
    - pattern: "*.txt"
      run: "` + scriptPath + `"
  on_url_found:
    - run: "` + scriptPath + `"
  on_collection_complete:
    - run: "` + scriptPath + `"
  on_error:
    - run: "` + scriptPath + `"
`
	configPath := filepath.Join(tmpDir, ".borg-hooks.yaml")
	err = os.WriteFile(configPath, []byte(hooksYAML), 0644)
	require.NoError(t, err)

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

func TestNewHookRunner(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	configPath := filepath.Join(tmpDir, ".borg-hooks.yaml")

	// Test with a valid config file
	runner, err := NewHookRunner(configPath)
	require.NoError(t, err)
	assert.NotNil(t, runner)
	assert.NotNil(t, runner.config)

	// Test with a non-existent file
	_, err = NewHookRunner("non-existent-file.yaml")
	assert.Error(t, err)

	// Test with a malformed file
	malformedConfigPath := filepath.Join(tmpDir, "malformed.yaml")
	err = os.WriteFile(malformedConfigPath, []byte("hooks: \n  - invalid"), 0644)
	require.NoError(t, err)
	_, err = NewHookRunner(malformedConfigPath)
	assert.Error(t, err)

	// Test with an empty config file path (should not error)
	runner, err = NewHookRunner("")
	require.NoError(t, err)
	assert.NotNil(t, runner)
	assert.NotNil(t, runner.config)
}

func TestHookRunner_Trigger(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	configPath := filepath.Join(tmpDir, ".borg-hooks.yaml")
	runner, err := NewHookRunner(configPath)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "testhook.output")

	tests := []struct {
		name          string
		event         Event
		shouldTrigger bool
		expectedEvent Event
	}{
		{
			name: "OnFileCollected - PDF Match with full path",
			event: Event{
				Event: OnFileCollected,
				File:  "assets/document.pdf",
				URL:   "http://example.com/assets/document.pdf",
				Type:  "application/pdf",
			},
			shouldTrigger: true,
			expectedEvent: Event{
				Event: OnFileCollected,
				File:  "assets/document.pdf",
				URL:   "http://example.com/assets/document.pdf",
				Type:  "application/pdf",
			},
		},
		{
			name: "OnFileCollected - TXT Match with full path",
			event: Event{
				Event: OnFileCollected,
				File:  "notes/notes.txt",
			},
			shouldTrigger: true,
			expectedEvent: Event{
				Event: OnFileCollected,
				File:  "notes/notes.txt",
			},
		},
		{
			name: "OnFileCollected - No Match",
			event: Event{
				Event: OnFileCollected,
				File:  "image.jpg",
			},
			shouldTrigger: false,
		},
		{
			name: "OnURLFound",
			event: Event{
				Event: OnURLFound,
				URL:   "http://example.com/page2",
			},
			shouldTrigger: true,
			expectedEvent: Event{
				Event: OnURLFound,
				URL:   "http://example.com/page2",
			},
		},
		{
			name: "OnCollectionComplete",
			event: Event{
				Event: OnCollectionComplete,
			},
			shouldTrigger: true,
			expectedEvent: Event{
				Event: OnCollectionComplete,
			},
		},
		{
			name: "OnError",
			event: Event{
				Event: OnError,
				Error: "something went wrong",
			},
			shouldTrigger: true,
			expectedEvent: Event{
				Event: OnError,
				Error: "something went wrong",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up previous output
			_ = os.Remove(outputFile)

			err := runner.Trigger(tt.event)
			require.NoError(t, err)

			if !tt.shouldTrigger {
				_, err := os.Stat(outputFile)
				assert.True(t, os.IsNotExist(err), "Hook should not have been triggered")
				return
			}

			// Verify the output file was created and contains the correct JSON
			content, err := os.ReadFile(outputFile)
			require.NoError(t, err)

			var receivedEvent Event
			err = json.Unmarshal(content, &receivedEvent)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedEvent, receivedEvent)
		})
	}
}

func TestHookRunner_FailingHook(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()

	// Create a failing script
	scriptContent := "#!/bin/sh\nexit 1"
	scriptPath := filepath.Join(tmpDir, "failing-hook.sh")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// Create hooks config with the failing script
	hooksYAML := `
hooks:
  on_error:
    - run: "` + scriptPath + `"
`
	configPath := filepath.Join(tmpDir, "failing-hooks.yaml")
	err = os.WriteFile(configPath, []byte(hooksYAML), 0644)
	require.NoError(t, err)

	runner, err := NewHookRunner(configPath)
	require.NoError(t, err)

	err = runner.Trigger(Event{Event: OnError, Error: "test error"})
	assert.Error(t, err)
}

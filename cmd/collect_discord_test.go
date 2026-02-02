package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Snider/Borg/pkg/mocks"
)

func TestCollectDiscordImportCmd_Good(t *testing.T) {
	// Mock HTTP client
	mockClient := mocks.NewMockClient(map[string]*http.Response{
		"https://example.com/file.txt": {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("attachment content")),
		},
	})
	http.DefaultClient = mockClient

	// Create a temporary directory
	tempDir := t.TempDir()

	// Read the sample export from testdata
	sampleData, err := os.ReadFile("testdata/sample_export.json")
	if err != nil {
		t.Fatalf("failed to read sample export file: %v", err)
	}
	jsonPath := filepath.Join(tempDir, "export.json")
	if err := os.WriteFile(jsonPath, sampleData, 0644); err != nil {
		t.Fatalf("failed to write sample json: %v", err)
	}

	// Change working directory to tempDir to check relative output path
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())

	// Execute command
	_, err = executeCommand(rootCmd, "collect", "discord", "import", "export.json")
	if err != nil {
		t.Fatalf("collect discord import command failed: %v", err)
	}

	// Verify output
	sanitizedServerName := "Test-Server"
	expectedBaseDir := filepath.Join("discord", sanitizedServerName)

	// Verify INDEX.json
	indexPath := filepath.Join(expectedBaseDir, "INDEX.json")
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read INDEX.json: %v", err)
	}
	type SearchEntry struct {
		ID        string    `json:"id"`
		Channel   string    `json:"channel"`
		Author    string    `json:"author"`
		Timestamp time.Time `json:"timestamp"`
		Content   string    `json:"content"`
	}
	var index []SearchEntry
	if err := json.Unmarshal(indexContent, &index); err != nil {
		t.Fatalf("failed to unmarshal INDEX.json: %v", err)
	}
	if len(index) != 3 {
		t.Fatalf("expected 3 messages in index, got %d", len(index))
	}
	if index[1].Content != "This is a test message." {
		t.Errorf("unexpected content in index entry: %s", index[1].Content)
	}

	// Verify attachment
	attachmentPath := filepath.Join(expectedBaseDir, "attachments", "file with spaces.txt")
	attachmentContent, err := os.ReadFile(attachmentPath)
	if err != nil {
		t.Fatalf("failed to read attachment: %v", err)
	}
	if string(attachmentContent) != "attachment content" {
		t.Errorf("unexpected content in attachment. Got: %s", string(attachmentContent))
	}

	// Verify random.md
	randomMdPath := filepath.Join(expectedBaseDir, "channels", "random.md")
	randomMdContent, err := os.ReadFile(randomMdPath)
	if err != nil {
		t.Fatalf("failed to read random.md: %v", err)
	}
	expectedRandomContent := "# random\n\n---\n**User2** `2024-01-01 12:01:00`\n\nThis is a test message.\n\n[file with spaces.txt](../attachments/file with spaces.txt)\n\n"
	if string(randomMdContent) != expectedRandomContent {
		t.Errorf("unexpected content in random.md.\nGot:\n%s\nExpected:\n%s", string(randomMdContent), expectedRandomContent)
	}
}

func TestCollectDiscordImportCmd_Bad(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())

	// Execute command with non-existent file
	_, err := executeCommand(rootCmd, "collect", "discord", "import", "non-existent.json")
	if err == nil {
		t.Fatal("expected an error, but got none")
	}
	if !strings.Contains(err.Error(), "could not open file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCollectDiscordImportCmd_Ugly(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.AddCommand(GetCollectCmd())
	_, err := executeCommand(rootCmd, "collect", "discord", "import")
	if err == nil {
		t.Fatal("expected an error for no arguments, but got none")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
		t.Errorf("unexpected error message: %v", err)
	}
}

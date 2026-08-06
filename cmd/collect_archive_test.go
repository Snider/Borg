package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/archive"
)

func TestCollectArchiveItemCmd_E2E(t *testing.T) {
	tempDir := t.TempDir()
	archiveDir := filepath.Join(tempDir, "archive")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/metadata/") {
			fmt.Fprintln(w, `{"files": [{"name": "test.txt", "format": "Text"}, {"name": "image.jpg", "format": "JPEG"}]}`)
		} else if strings.Contains(r.URL.Path, "/download/") {
			fmt.Fprintln(w, "file content")
		}
	}))
	defer server.Close()

	originalURL := archive.BaseURL
	archive.BaseURL = server.URL
	defer func() {
		archive.BaseURL = originalURL
	}()

	// Change working directory for the test
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	_, err := executeCommand(RootCmd, "collect", "archive", "item", "test-item")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory and files
	itemDir := filepath.Join(archiveDir, "test-item")
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		t.Errorf("expected directory %s to be created", itemDir)
	}

	for _, f := range []string{"metadata.json", "_files.json", "test.txt", "image.jpg"} {
		if _, err := os.Stat(filepath.Join(itemDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %s to be created in %s", f, itemDir)
		}
	}
}

func TestCollectArchiveSearchCmd_FormatFlag_E2E(t *testing.T) {
	tempDir := t.TempDir()
	archiveDir := filepath.Join(tempDir, "archive")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/advancedsearch.php") {
			fmt.Fprintln(w, `{"response": {"docs": [{"identifier": "test-item"}]}}`)
		} else if strings.Contains(r.URL.Path, "/metadata/") {
			fmt.Fprintln(w, `{"files": [{"name": "test.txt", "format": "Text"}, {"name": "image.jpg", "format": "JPEG"}]}`)
		} else if strings.Contains(r.URL.Path, "/download/") {
			fmt.Fprintln(w, "file content")
		}
	}))
	defer server.Close()

	originalURL := archive.BaseURL
	archive.BaseURL = server.URL
	defer func() {
		archive.BaseURL = originalURL
	}()

	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	_, err := executeCommand(RootCmd, "collect", "archive", "search", "test-query", "--format=Text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	itemDir := filepath.Join(archiveDir, "test-item")
	// Verify correct file is downloaded
	if _, err := os.Stat(filepath.Join(itemDir, "test.txt")); os.IsNotExist(err) {
		t.Errorf("expected test.txt to be downloaded")
	}
	// Verify incorrect format file is NOT downloaded
	if _, err := os.Stat(filepath.Join(itemDir, "image.jpg")); err == nil {
		t.Errorf("did not expect image.jpg to be downloaded")
	}
}

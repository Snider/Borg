package cmd

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/mocks"
	borg_github "github.com/Snider/Borg/pkg/github"
	"github.com/google/go-github/v39/github"
)

func TestGetRelease_Good(t *testing.T) {
	// Create a temporary directory for the output
	dir, err := os.MkdirTemp("", "test-get-release")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	mockClient := mocks.NewMockClient(map[string]*http.Response{
		"https://api.github.com/repos/owner/repo/releases/latest": {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"tag_name": "v1.0.0", "assets": [{"name": "asset1.zip", "browser_download_url": "https://github.com/owner/repo/releases/download/v1.0.0/asset1.zip"}]}`)),
		},
		"https://github.com/owner/repo/releases/download/v1.0.0/asset1.zip": {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("asset content")),
		},
	})

	oldNewClient := borg_github.NewClient
	borg_github.NewClient = func(httpClient *http.Client) *github.Client {
		return github.NewClient(mockClient)
	}
	defer func() {
		borg_github.NewClient = oldNewClient
	}()

	oldDefaultClient := borg_github.DefaultClient
	borg_github.DefaultClient = mockClient
	defer func() {
		borg_github.DefaultClient = oldDefaultClient
	}()

	log := slog.New(slog.NewJSONHandler(io.Discard, nil))

	// Test downloading a single asset
	_, err = GetRelease(log, "https://github.com/owner/repo", dir, false, "asset1.zip", "")
	if err != nil {
		t.Fatalf("GetRelease failed: %v", err)
	}

	// Verify the asset was downloaded
	content, err := os.ReadFile(filepath.Join(dir, "asset1.zip"))
	if err != nil {
		t.Fatalf("failed to read downloaded asset: %v", err)
	}
	if string(content) != "asset content" {
		t.Errorf("unexpected asset content: %s", string(content))
	}

	// Test packing all assets
	packedDir := filepath.Join(dir, "packed")
	_, err = GetRelease(log, "https://github.com/owner/repo", packedDir, true, "", "")
	if err != nil {
		t.Fatalf("GetRelease with --pack failed: %v", err)
	}

	// Verify the datanode was created
	if _, err := os.Stat(filepath.Join(packedDir, "v1.0.0.dat")); os.IsNotExist(err) {
		t.Fatalf("datanode not created")
	}
}

func TestGetRelease_Bad(t *testing.T) {
	// Create a temporary directory for the output
	dir, err := os.MkdirTemp("", "test-get-release")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	mockClient := mocks.NewMockClient(map[string]*http.Response{
		"https://api.github.com/repos/owner/repo/releases/latest": {
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Not Found"}`)),
		},
	})

	oldNewClient := borg_github.NewClient
	borg_github.NewClient = func(httpClient *http.Client) *github.Client {
		return github.NewClient(mockClient)
	}
	defer func() {
		borg_github.NewClient = oldNewClient
	}()

	log := slog.New(slog.NewJSONHandler(io.Discard, nil))

	// Test failed release lookup
	_, err = GetRelease(log, "https://github.com/owner/repo", dir, false, "", "")
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}
}

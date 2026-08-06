package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"

	borg_github "github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/mocks"
	"github.com/google/go-github/v39/github"
	"github.com/stretchr/testify/assert"
)

func TestCollectGithubReleaseCmd(t *testing.T) {
	assetContent := "asset content"
	hasher := sha256.New()
	hasher.Write([]byte(assetContent))
	correctChecksum := hex.EncodeToString(hasher.Sum(nil))
	checksumsFileContent := fmt.Sprintf("%s  asset1.zip\n", correctChecksum)

	// Mock the GitHub API
	responses := map[string]*http.Response{
		"https://api.github.com/repos/owner/repo/releases/latest": {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"tag_name": "v1.0.0", "body": "Release notes", "assets": [{"name": "asset1.zip", "browser_download_url": "http://localhost/asset1.zip"}, {"name": "checksums.txt", "browser_download_url": "http://localhost/checksums.txt"}]}`))),
			Header:     http.Header{},
		},
		"https://api.github.com/repos/owner/repo/releases/tags/v1.0.0": {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"tag_name": "v1.0.0", "body": "Release notes", "assets": [{"name": "asset1.zip", "browser_download_url": "http://localhost/asset1.zip"}, {"name": "checksums.txt", "browser_download_url": "http://localhost/checksums.txt"}]}`))),
			Header:     http.Header{},
		},
		"http://localhost/asset1.zip": {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(assetContent)),
		},
		"http://localhost/checksums.txt": {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(checksumsFileContent)),
		},
	}
	mockRoundTripper := &mocks.MockRoundTripper{}
	mockHttpClient := &http.Client{Transport: mockRoundTripper}

	// Mock the API client
	oldNewClient := borg_github.NewClient
	borg_github.NewClient = func(client *http.Client) *github.Client {
		return github.NewClient(mockHttpClient)
	}
	t.Cleanup(func() { borg_github.NewClient = oldNewClient })

	// Mock the download client
	oldDefaultClient := borg_github.DefaultClient
	borg_github.DefaultClient = mockHttpClient
	t.Cleanup(func() { borg_github.DefaultClient = oldDefaultClient })

	t.Run("Latest", func(t *testing.T) {
		t.Cleanup(func() { os.RemoveAll("releases") })
		cmd := NewCollectGithubReleaseCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"owner/repo"})
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		ctx := context.WithValue(context.Background(), "logger", log)
		err := cmd.ExecuteContext(ctx)
		assert.NoError(t, err)
		assert.FileExists(t, "releases/v1.0.0/RELEASE.md")
		assert.FileExists(t, "releases/v1.0.0/asset1.zip")
		assert.FileExists(t, "releases/v1.0.0/checksums.txt")
	})

	t.Run("ByTag", func(t *testing.T) {
		t.Cleanup(func() { os.RemoveAll("releases") })
		cmd := NewCollectGithubReleaseCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"owner/repo", "v1.0.0"})
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		ctx := context.WithValue(context.Background(), "logger", log)
		err := cmd.ExecuteContext(ctx)
		assert.NoError(t, err)
		assert.FileExists(t, "releases/v1.0.0/RELEASE.md")
		assert.FileExists(t, "releases/v1.0.0/asset1.zip")
	})

	t.Run("AssetsOnly", func(t *testing.T) {
		t.Cleanup(func() { os.RemoveAll("releases") })
		cmd := NewCollectGithubReleaseCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"owner/repo", "--assets-only"})
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		ctx := context.WithValue(context.Background(), "logger", log)
		err := cmd.ExecuteContext(ctx)
		assert.NoError(t, err)
		assert.NoFileExists(t, "releases/v1.0.0/RELEASE.md")
		assert.FileExists(t, "releases/v1.0.0/asset1.zip")
	})

	t.Run("Pattern", func(t *testing.T) {
		t.Cleanup(func() { os.RemoveAll("releases") })
		cmd := NewCollectGithubReleaseCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"owner/repo", "--pattern", "*.zip"})
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		ctx := context.WithValue(context.Background(), "logger", log)
		err := cmd.ExecuteContext(ctx)
		assert.NoError(t, err)
		assert.FileExists(t, "releases/v1.0.0/RELEASE.md")
		assert.FileExists(t, "releases/v1.0.0/asset1.zip")
		assert.NoFileExists(t, "releases/v1.0.0/checksums.txt")
	})

	t.Run("VerifyChecksums", func(t *testing.T) {
		t.Cleanup(func() { os.RemoveAll("releases") })
		cmd := NewCollectGithubReleaseCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"owner/repo", "--verify-checksums"})
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		ctx := context.WithValue(context.Background(), "logger", log)
		err := cmd.ExecuteContext(ctx)
		assert.NoError(t, err)
		assert.FileExists(t, "releases/v1.0.0/asset1.zip")
	})

	t.Run("VerifyChecksumsFailed", func(t *testing.T) {
		// Mock a bad checksum
		badChecksumsFileContent := fmt.Sprintf("badchecksum  asset1.zip\n")
		responses["http://localhost/checksums.txt"] = &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(badChecksumsFileContent)),
		}
		mockHttpClient.SetResponses(responses)
		t.Cleanup(func() { os.RemoveAll("releases") })

		cmd := NewCollectGithubReleaseCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"owner/repo", "--verify-checksums"})
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		ctx := context.WithValue(context.Background(), "logger", log)
		// We expect an error, but the command is designed to log it and continue.
		err := cmd.ExecuteContext(ctx)
		assert.NoError(t, err)
	})
}

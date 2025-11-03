package github

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/Snider/Borg/pkg/mocks"
	"github.com/google/go-github/v39/github"
)

type errorRoundTripper struct{}

func (e *errorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("do error")
}

func TestParseRepoFromURL(t *testing.T) {
	testCases := []struct {
		url       string
		owner     string
		repo      string
		expectErr bool
	}{
		{"https://github.com/owner/repo.git", "owner", "repo", false},
		{"http://github.com/owner/repo", "owner", "repo", false},
		{"git://github.com/owner/repo.git", "owner", "repo", false},
		{"github.com/owner/repo", "owner", "repo", false},
		{"git@github.com:owner/repo.git", "owner", "repo", false},
		{"https://github.com/owner/repo/tree/main", "", "", true},
		{"invalid-url", "", "", true},
	}

	for _, tc := range testCases {
		owner, repo, err := ParseRepoFromURL(tc.url)
		if (err != nil) != tc.expectErr {
			t.Errorf("unexpected error for URL %s: %v", tc.url, err)
		}
		if owner != tc.owner || repo != tc.repo {
			t.Errorf("unexpected owner/repo for URL %s: %s/%s", tc.url, owner, repo)
		}
	}
}

func TestGetLatestRelease(t *testing.T) {
	oldNewClient := NewClient
	t.Cleanup(func() { NewClient = oldNewClient })

	mockClient := mocks.NewMockClient(map[string]*http.Response{
		"https://api.github.com/repos/owner/repo/releases/latest": {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"tag_name": "v1.0.0"}`)),
		},
	})
	client := github.NewClient(mockClient)
	NewClient = func(_ *http.Client) *github.Client {
		return client
	}

	release, err := GetLatestRelease("owner", "repo")
	if err != nil {
		t.Fatalf("GetLatestRelease failed: %v", err)
	}

	if release.GetTagName() != "v1.0.0" {
		t.Errorf("unexpected tag name: %s", release.GetTagName())
	}
}

func TestDownloadReleaseAsset(t *testing.T) {
	mockClient := mocks.NewMockClient(map[string]*http.Response{
		"https://github.com/owner/repo/releases/download/v1.0.0/asset.zip": {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("asset content")),
		},
	})

	asset := &github.ReleaseAsset{
		BrowserDownloadURL: github.String("https://github.com/owner/repo/releases/download/v1.0.0/asset.zip"),
	}

	oldClient := DefaultClient
	DefaultClient = mockClient
	defer func() {
		DefaultClient = oldClient
	}()

	data, err := DownloadReleaseAsset(asset)
	if err != nil {
		t.Fatalf("DownloadReleaseAsset failed: %v", err)
	}

	if string(data) != "asset content" {
		t.Errorf("unexpected asset content: %s", string(data))
	}
}
func TestDownloadReleaseAsset_BadRequest(t *testing.T) {
	mockClient := mocks.NewMockClient(map[string]*http.Response{
		"https://github.com/owner/repo/releases/download/v1.0.0/asset.zip": {
			StatusCode: http.StatusBadRequest,
			Status:     "400 Bad Request",
			Body:       io.NopCloser(bytes.NewBufferString("")),
		},
	})
	expectedErr := "bad status: 400 Bad Request"

	asset := &github.ReleaseAsset{
		BrowserDownloadURL: github.String("https://github.com/owner/repo/releases/download/v1.0.0/asset.zip"),
	}

	oldClient := DefaultClient
	DefaultClient = mockClient
	defer func() {
		DefaultClient = oldClient
	}()

	_, err := DownloadReleaseAsset(asset)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if err.Error() != expectedErr {
		t.Fatalf("DownloadReleaseAsset failed: %v", err)
	}
}

func TestDownloadReleaseAsset_NewRequestError(t *testing.T) {
	errRequest := fmt.Errorf("bad request")
	asset := &github.ReleaseAsset{
		BrowserDownloadURL: github.String("https://github.com/owner/repo/releases/download/v1.0.0/asset.zip"),
	}

	oldNewRequest := NewRequest
	NewRequest = func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, errRequest
	}
	defer func() {
		NewRequest = oldNewRequest
	}()

	_, err := DownloadReleaseAsset(asset)
	if err == nil {
		t.Fatalf("DownloadReleaseAsset failed: %v", err)
	}
}

func TestGetLatestRelease_Error(t *testing.T) {
	oldNewClient := NewClient
	t.Cleanup(func() { NewClient = oldNewClient })

	u, _ := url.Parse("https://api.github.com/repos/owner/repo/releases/latest")
	mockClient := mocks.NewMockClient(map[string]*http.Response{
		"https://api.github.com/repos/owner/repo/releases/latest": {
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString("")),
			Request:    &http.Request{Method: "GET", URL: u},
		},
	})
	expectedErr := "GET https://api.github.com/repos/owner/repo/releases/latest: 404  []"
	client := github.NewClient(mockClient)
	NewClient = func(_ *http.Client) *github.Client {
		return client
	}

	_, err := GetLatestRelease("owner", "repo")
	if err.Error() != expectedErr {
		t.Fatalf("GetLatestRelease failed: %v", err)
	}
}

func TestDownloadReleaseAsset_DoError(t *testing.T) {
	mockClient := &http.Client{
		Transport: &errorRoundTripper{},
	}

	asset := &github.ReleaseAsset{
		BrowserDownloadURL: github.String("https://github.com/owner/repo/releases/download/v1.0.0/asset.zip"),
	}

	oldClient := DefaultClient
	DefaultClient = mockClient
	defer func() {
		DefaultClient = oldClient
	}()

	_, err := DownloadReleaseAsset(asset)
	if err == nil {
		t.Fatalf("DownloadReleaseAsset should have failed")
	}
}
func TestParseRepoFromURL_More(t *testing.T) {
	testCases := []struct {
		url       string
		owner     string
		repo      string
		expectErr bool
	}{
		{"git:github.com:owner/repo.git", "owner", "repo", false},
	}

	for _, tc := range testCases {
		owner, repo, err := ParseRepoFromURL(tc.url)
		if (err != nil) != tc.expectErr {
			t.Errorf("unexpected error for URL %s: %v", tc.url, err)
		}
		if owner != tc.owner || repo != tc.repo {
			t.Errorf("unexpected owner/repo for URL %s: %s/%s", tc.url, owner, repo)
		}
	}
}

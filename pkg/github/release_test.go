package github

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/Snider/Borg/pkg/mocks"
	"github.com/google/go-github/v39/github"
	"github.com/stretchr/testify/assert"
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

	buf := new(bytes.Buffer)
	err := DownloadReleaseAsset(asset, buf)
	if err != nil {
		t.Fatalf("DownloadReleaseAsset failed: %v", err)
	}

	if buf.String() != "asset content" {
		t.Errorf("unexpected asset content: %s", buf.String())
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

	buf := new(bytes.Buffer)
	err := DownloadReleaseAsset(asset, buf)
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

	buf := new(bytes.Buffer)
	err := DownloadReleaseAsset(asset, buf)
	if err == nil {
		t.Fatalf("expected error but got nil")
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
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
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

	buf := new(bytes.Buffer)
	err := DownloadReleaseAsset(asset, buf)
	if err == nil {
		t.Fatalf("DownloadReleaseAsset should have failed")
	}
}

func TestListReleases(t *testing.T) {
	oldNewClient := NewClient
	t.Cleanup(func() { NewClient = oldNewClient })

	responses := make(map[string]*http.Response)
	responses["https://api.github.com/repos/owner/repo/releases?per_page=30"] = &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`[{"tag_name": "v1.0.0"}, {"tag_name": "v0.9.0"}]`)),
		Header: http.Header{
			"Link": []string{`<https://api.github.com/repos/owner/repo/releases?page=2&per_page=30>; rel="next"`},
		},
	}
	responses["https://api.github.com/repos/owner/repo/releases?page=2&per_page=30"] = &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`[{"tag_name": "v0.8.0"}]`)),
		Header:     http.Header{},
	}

	mockClient := mocks.NewMockClient(responses)
	client := github.NewClient(mockClient)
	NewClient = func(_ *http.Client) *github.Client {
		return client
	}

	releases, err := ListReleases("owner", "repo")
	assert.NoError(t, err)
	assert.Len(t, releases, 3)
	assert.Equal(t, "v1.0.0", releases[0].GetTagName())
	assert.Equal(t, "v0.9.0", releases[1].GetTagName())
	assert.Equal(t, "v0.8.0", releases[2].GetTagName())
}

func TestGetReleaseByTag(t *testing.T) {
	oldNewClient := NewClient
	t.Cleanup(func() { NewClient = oldNewClient })

	mockClient := mocks.NewMockClient(map[string]*http.Response{
		"https://api.github.com/repos/owner/repo/releases/tags/v1.0.0": {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"tag_name": "v1.0.0"}`)),
		},
	})
	client := github.NewClient(mockClient)
	NewClient = func(_ *http.Client) *github.Client {
		return client
	}

	release, err := GetReleaseByTag("owner", "repo", "v1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, release)
	assert.Equal(t, "v1.0.0", release.GetTagName())
}

func TestDownloadReleaseAssetWithChecksum(t *testing.T) {
	assetName := "my-asset.zip"
	assetURL := "https://example.com/download/my-asset.zip"
	assetContent := "this is the content of the asset"
	hasher := sha256.New()
	hasher.Write([]byte(assetContent))
	correctChecksum := hex.EncodeToString(hasher.Sum(nil))
	checksumsFileContent := fmt.Sprintf("%s  %s\n", correctChecksum, assetName)

	asset := &github.ReleaseAsset{
		Name:               &assetName,
		BrowserDownloadURL: &assetURL,
	}

	t.Run("GoodChecksum", func(t *testing.T) {
		mockHttpClient := mocks.NewMockClient(map[string]*http.Response{
			assetURL: {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(assetContent)),
			},
		})
		oldClient := DefaultClient
		DefaultClient = mockHttpClient
		t.Cleanup(func() { DefaultClient = oldClient })

		buf := new(bytes.Buffer)
		err := DownloadReleaseAssetWithChecksum(asset, []byte(checksumsFileContent), buf)
		assert.NoError(t, err)
		assert.Equal(t, assetContent, buf.String())
	})

	t.Run("BadChecksum", func(t *testing.T) {
		mockHttpClient := mocks.NewMockClient(map[string]*http.Response{
			assetURL: {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(assetContent)),
			},
		})
		oldClient := DefaultClient
		DefaultClient = mockHttpClient
		t.Cleanup(func() { DefaultClient = oldClient })
		badChecksumsFileContent := fmt.Sprintf("badchecksum  %s\n", assetName)
		buf := new(bytes.Buffer)
		err := DownloadReleaseAssetWithChecksum(asset, []byte(badChecksumsFileContent), buf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")
	})

	t.Run("ChecksumNotFound", func(t *testing.T) {
		mockHttpClient := mocks.NewMockClient(map[string]*http.Response{
			assetURL: {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(assetContent)),
			},
		})
		oldClient := DefaultClient
		DefaultClient = mockHttpClient
		t.Cleanup(func() { DefaultClient = oldClient })
		missingChecksumsFileContent := fmt.Sprintf("%s  another-file.zip\n", correctChecksum)
		buf := new(bytes.Buffer)
		err := DownloadReleaseAssetWithChecksum(asset, []byte(missingChecksumsFileContent), buf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checksum not found")
	})

	t.Run("DownloadError", func(t *testing.T) {
		mockHttpClientWithErr := mocks.NewMockClient(map[string]*http.Response{
			assetURL: {
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(bytes.NewBufferString("")),
			},
		})
		oldClient := DefaultClient
		DefaultClient = mockHttpClientWithErr
		t.Cleanup(func() { DefaultClient = oldClient })

		buf := new(bytes.Buffer)
		err := DownloadReleaseAssetWithChecksum(asset, []byte(checksumsFileContent), buf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bad status: 404 Not Found")
	})
}

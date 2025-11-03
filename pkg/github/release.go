package github

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/v39/github"
)

var (
	// NewClient is a variable that holds the function to create a new GitHub client.
	// This allows for mocking in tests.
	NewClient = func(httpClient *http.Client) *github.Client {
		return github.NewClient(httpClient)
	}
	// NewRequest is a variable that holds the function to create a new HTTP request.
	NewRequest = func(method, url string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, url, body)
	}
	// DefaultClient is the default http client
	DefaultClient = &http.Client{}
)

// GetLatestRelease gets the latest release for a repository.
func GetLatestRelease(owner, repo string) (*github.RepositoryRelease, error) {
	client := NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		return nil, err
	}
	return release, nil
}

// DownloadReleaseAsset downloads a release asset.
func DownloadReleaseAsset(asset *github.ReleaseAsset) ([]byte, error) {
	req, err := NewRequest("GET", asset.GetBrowserDownloadURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ParseRepoFromURL parses the owner and repository from a GitHub URL.
func ParseRepoFromURL(u string) (owner, repo string, err error) {
	u = strings.TrimSuffix(u, ".git")

	prefixesToTrim := []string{
		"https://github.com/",
		"http://github.com/",
		"git://github.com/",
		"github.com/",
	}

	// Handle scp-like and other formats by replacing them first.
	u = strings.Replace(u, "git@github.com:", "", 1)
	u = strings.Replace(u, "git:github.com:", "", 1)

	for _, p := range prefixesToTrim {
		if strings.HasPrefix(u, p) {
			u = strings.TrimPrefix(u, p)
			break
		}
	}

	parts := strings.Split(u, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid or unsupported github url format: %s", u)
	}

	return parts[0], parts[1], nil
}

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
	// NewClient is a function that creates a new GitHub client. It is a
	// variable to allow for mocking in tests.
	NewClient = func(httpClient *http.Client) *github.Client {
		return github.NewClient(httpClient)
	}
	// NewRequest is a function that creates a new HTTP request. It is a
	// variable to allow for mocking in tests.
	NewRequest = func(method, url string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, url, body)
	}
	// DefaultClient is the default http client used for making requests. It is
	// a variable to allow for mocking in tests.
	DefaultClient = &http.Client{}
)

// GetLatestRelease fetches the latest release metadata for a given GitHub
// repository.
//
// Example:
//
//	release, err := github.GetLatestRelease("my-org", "my-repo")
//	if err != nil {
//		// handle error
//	}
//	fmt.Println(release.GetTagName())
func GetLatestRelease(owner, repo string) (*github.RepositoryRelease, error) {
	client := NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		return nil, err
	}
	return release, nil
}

// DownloadReleaseAsset downloads the content of a release asset.
//
// Example:
//
//	// Assuming 'release' is a *github.RepositoryRelease
//	for _, asset := range release.Assets {
//		if asset.GetName() == "my-asset.zip" {
//			data, err := github.DownloadReleaseAsset(asset)
//			if err != nil {
//				// handle error
//			}
//			// do something with data
//		}
//	}
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

// ParseRepoFromURL extracts the owner and repository name from a variety of
// GitHub URL formats, including HTTPS, Git, and SCP-style URLs.
//
// Example:
//
//	owner, repo, err := github.ParseRepoFromURL("https://github.com/my-org/my-repo.git")
//	if err != nil {
//		// handle error
//	}
//	fmt.Println(owner, repo) // "my-org", "my-repo"
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

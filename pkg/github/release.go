package github

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// DownloadReleaseAssetWithChecksum downloads a release asset and verifies its checksum.
func DownloadReleaseAssetWithChecksum(asset *github.ReleaseAsset, checksumsData []byte, w io.Writer) error {
	req, err := NewRequest("GET", asset.GetBrowserDownloadURL(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	hasher := sha256.New()
	teeReader := io.TeeReader(resp.Body, hasher)

	_, err = io.Copy(w, teeReader)
	if err != nil {
		return err
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))

	err = verifyChecksum(actualChecksum, asset.GetName(), checksumsData)
	if err != nil {
		return fmt.Errorf("checksum verification failed for %s: %w", asset.GetName(), err)
	}

	return nil
}

// verifyChecksum verifies the SHA256 checksum of a byte slice against a checksums file content.
// The checksums file is expected to be in the format: <checksum>  <filename>
func verifyChecksum(actualChecksum, name string, checksumsData []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(checksumsData))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == name {
			expectedChecksum := parts[0]
			if actualChecksum != expectedChecksum {
				return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
			}
			return nil // Checksum verified
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading checksums data: %w", err)
	}

	return fmt.Errorf("checksum not found for file: %s", name)
}

// ListReleases lists all releases for a repository.
func ListReleases(owner, repo string) ([]*github.RepositoryRelease, error) {
	client := NewClient(nil)
	opt := &github.ListOptions{PerPage: 30}
	var allReleases []*github.RepositoryRelease
	for {
		releases, resp, err := client.Repositories.ListReleases(context.Background(), owner, repo, opt)
		if err != nil {
			return nil, err
		}
		allReleases = append(allReleases, releases...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allReleases, nil
}

// GetReleaseByTag gets a release by its tag name.
func GetReleaseByTag(owner, repo, tag string) (*github.RepositoryRelease, error) {
	client := NewClient(nil)
	release, _, err := client.Repositories.GetReleaseByTag(context.Background(), owner, repo, tag)
	if err != nil {
		return nil, err
	}
	return release, nil
}

// DownloadReleaseAsset downloads a release asset.
func DownloadReleaseAsset(asset *github.ReleaseAsset, w io.Writer) error {
	req, err := NewRequest("GET", asset.GetBrowserDownloadURL(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(w, resp.Body)
	return err
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

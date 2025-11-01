package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v39/github"
)

// GetLatestRelease gets the latest release for a repository.
func GetLatestRelease(owner, repo string) (*github.RepositoryRelease, error) {
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		return nil, err
	}
	return release, nil
}

// DownloadReleaseAsset downloads a release asset.
func DownloadReleaseAsset(asset *github.ReleaseAsset, path string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", asset.GetBrowserDownloadURL(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
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

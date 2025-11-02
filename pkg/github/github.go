package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Snider/Borg/pkg/mocks"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
)

type Repo struct {
	CloneURL string `json:"clone_url"`
}

func GetPublicRepos(ctx context.Context, userOrOrg string) ([]string, error) {
	return GetPublicReposWithAPIURL(ctx, "https://api.github.com", userOrOrg)
}

func newAuthenticatedClient(ctx context.Context) *http.Client {
	if os.Getenv("BORG_PLEXSUS") == "0" {
		// Define mock responses for testing
		responses := map[string]*http.Response{
			"https://api.github.com/users/test/repos": {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`[{"clone_url": "https://github.com/test/repo1.git"}]`)),
				Header:     make(http.Header),
			},
			"https://api.github.com/orgs/test/repos": {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`[{"clone_url": "https://github.com/test/repo2.git"}]`)),
				Header:     make(http.Header),
			},
		}
		return mocks.NewMockClient(responses)
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return http.DefaultClient
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	return oauth2.NewClient(ctx, ts)
}

func GetPublicReposWithAPIURL(ctx context.Context, apiURL, userOrOrg string) ([]string, error) {
	client := newAuthenticatedClient(ctx)
	var allCloneURLs []string
	url := fmt.Sprintf("%s/users/%s/repos", apiURL, userOrOrg)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Borg-Data-Collector")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			// Try organization endpoint
			url = fmt.Sprintf("%s/orgs/%s/repos", apiURL, userOrOrg)
			req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("User-Agent", "Borg-Data-Collector")
			resp, err = client.Do(req)
			if err != nil {
				return nil, err
			}
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch repos: %s", resp.Status)
		}

		var repos []Repo
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, repo := range repos {
			allCloneURLs = append(allCloneURLs, repo.CloneURL)
		}

		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" {
			break
		}
		nextURL := findNextURL(linkHeader)
		if nextURL == "" {
			break
		}
		url = nextURL
	}

	return allCloneURLs, nil
}

func findNextURL(linkHeader string) string {
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(link, ";")
		if len(parts) == 2 && strings.TrimSpace(parts[1]) == `rel="next"` {
			return strings.Trim(strings.TrimSpace(parts[0]), "<>")
		}
	}
	return ""
}

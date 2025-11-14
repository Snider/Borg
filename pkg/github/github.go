package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
)

type Repo struct {
	CloneURL string `json:"clone_url"`
}

// GithubClient is an interface for interacting with the Github API.
type GithubClient interface {
	GetPublicRepos(ctx context.Context, userOrOrg string) ([]string, error)
}

// NewGithubClient creates a new GithubClient.
func NewGithubClient() GithubClient {
	return &githubClient{}
}

type githubClient struct{}

// NewAuthenticatedClient creates a new authenticated http client.
var NewAuthenticatedClient = func(ctx context.Context) *http.Client {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return http.DefaultClient
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	return oauth2.NewClient(ctx, ts)
}

func (g *githubClient) GetPublicRepos(ctx context.Context, userOrOrg string) ([]string, error) {
	return g.getPublicReposWithAPIURL(ctx, "https://api.github.com", userOrOrg)
}

func (g *githubClient) getPublicReposWithAPIURL(ctx context.Context, apiURL, userOrOrg string) ([]string, error) {
	client := NewAuthenticatedClient(ctx)
	var allCloneURLs []string
	url := fmt.Sprintf("%s/users/%s/repos", apiURL, userOrOrg)
	isFirstRequest := true

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
			// If it's the first request for a user and it's a 404, we can try the org endpoint.
			if isFirstRequest && strings.Contains(url, "/users/") && resp.StatusCode == http.StatusNotFound {
				resp.Body.Close()
				url = fmt.Sprintf("%s/orgs/%s/repos", apiURL, userOrOrg)
				isFirstRequest = false // We are now trying the org endpoint.
				continue                 // Re-run the loop with the org URL.
			}
			status := resp.Status
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch repos: %s", status)
		}

		isFirstRequest = false // Subsequent requests are for pagination.

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
		nextURL := g.findNextURL(linkHeader)
		if nextURL == "" {
			break
		}
		url = nextURL
	}

	return allCloneURLs, nil
}

func (g *githubClient) findNextURL(linkHeader string) string {
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(link, ";")
		if len(parts) < 2 {
			continue
		}

		if strings.TrimSpace(parts[1]) == `rel="next"` {
			urlPart := strings.TrimSpace(parts[0])
			if strings.HasPrefix(urlPart, "<") && strings.HasSuffix(urlPart, ">") {
				return urlPart[1 : len(urlPart)-1]
			}
		}
	}
	return ""
}

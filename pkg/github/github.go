package github

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Repo struct {
	CloneURL string `json:"clone_url"`
}

func GetPublicRepos(userOrOrg string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/users/%s/repos", userOrOrg))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try organization endpoint
		resp, err = http.Get(fmt.Sprintf("https://api.github.com/orgs/%s/repos", userOrOrg))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch repos: %s", resp.Status)
		}
	}

	var repos []Repo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	var cloneURLs []string
	for _, repo := range repos {
		cloneURLs = append(cloneURLs, repo.CloneURL)
	}

	return cloneURLs, nil
}

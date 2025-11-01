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
	return GetPublicReposWithAPIURL("https://api.github.com", userOrOrg)
}

func GetPublicReposWithAPIURL(apiURL, userOrOrg string) ([]string, error) {
	if userOrOrg == "" {
		return nil, fmt.Errorf("user or organization cannot be empty")
	}

	resp, err := http.Get(fmt.Sprintf("%s/users/%s/repos", apiURL, userOrOrg))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try organization endpoint
		resp, err = http.Get(fmt.Sprintf("%s/orgs/%s/repos", apiURL, userOrOrg))
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

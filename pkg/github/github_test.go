package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPublicRepos_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"clone_url": "https://github.com/good/repo.git"}]`))
	}))
	defer server.Close()

	repos, err := GetPublicReposWithAPIURL(server.URL, "good")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(repos) != 1 || repos[0] != "https://github.com/good/repo.git" {
		t.Errorf("Expected one repo, got %v", repos)
	}
}

func TestGetPublicRepos_Bad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := GetPublicReposWithAPIURL(server.URL, "bad")
	if err == nil {
		t.Errorf("Expected an error, got nil")
	}
}

func TestGetPublicRepos_Ugly(t *testing.T) {
	_, err := GetPublicReposWithAPIURL("http://localhost", "")
	if err == nil {
		t.Errorf("Expected an error for empty user/org, got nil")
	}
}

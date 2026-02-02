package github

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPullRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/pulls" {
			w.Header().Set("Content-Type", "application/json")
			prs := []PullRequest{
				{
					Number: 1, Title: "PR 1",
					DiffURL: "http://" + r.Host + "/repos/owner/repo/pulls/1.diff",
					Links: struct {
						Comments       struct{ Href string `json:"href"` } `json:"comments"`
						ReviewComments struct{ Href string `json:"href"` } `json:"review_comments"`
					}{
						ReviewComments: struct{ Href string `json:"href"` }{Href: "http://" + r.Host + "/repos/owner/repo/pulls/1/comments"},
					},
				},
				{
					Number: 2, Title: "PR 2",
					DiffURL: "http://" + r.Host + "/repos/owner/repo/pulls/2.diff",
					Links: struct {
						Comments       struct{ Href string `json:"href"` } `json:"comments"`
						ReviewComments struct{ Href string `json:"href"` } `json:"review_comments"`
					}{
						ReviewComments: struct{ Href string `json:"href"` }{Href: "http://" + r.Host + "/repos/owner/repo/pulls/2/comments"},
					},
				},
			}
			json.NewEncoder(w).Encode(prs)
		} else if r.URL.Path == "/repos/owner/repo/pulls/1.diff" {
			w.Write([]byte("diff --git a/file b/file"))
		} else if r.URL.Path == "/repos/owner/repo/pulls/1/comments" {
			w.Header().Set("Content-Type", "application/json")
			comments := []ReviewComment{
				{Body: "Review Comment 1"},
			}
			json.NewEncoder(w).Encode(comments)
		} else if r.URL.Path == "/repos/owner/repo/pulls/2.diff" {
			w.Write([]byte("diff --git a/file2 b/file2"))
		} else if r.URL.Path == "/repos/owner/repo/pulls/2/comments" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	originalNewAuthenticatedClient := NewAuthenticatedClient
	NewAuthenticatedClient = func(ctx context.Context) *http.Client {
		return server.Client()
	}
	defer func() {
		NewAuthenticatedClient = originalNewAuthenticatedClient
	}()

	client := &githubClient{apiURL: server.URL}
	dn, err := client.GetPullRequests(context.Background(), "owner", "repo")

	assert.NoError(t, err)
	assert.NotNil(t, dn)

	expectedFiles := []string{
		"pulls/1.md",
		"pulls/1.diff",
		"pulls/2.md",
		"pulls/2.diff",
		"pulls/INDEX.json",
	}

	actualFiles := []string{}
	dn.Walk(".", func(path string, de fs.DirEntry, err error) error {
		if !de.IsDir() {
			actualFiles = append(actualFiles, path)
		}
		return nil
	})

	assert.ElementsMatch(t, expectedFiles, actualFiles)
}

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

func TestGetIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/issues" {
			w.Header().Set("Content-Type", "application/json")
			issues := []Issue{
				{Number: 1, Title: "Issue 1", CommentsURL: "http://" + r.Host + "/repos/owner/repo/issues/1/comments"},
				{Number: 2, Title: "Issue 2", CommentsURL: "http://" + r.Host + "/repos/owner/repo/issues/2/comments"},
			}
			json.NewEncoder(w).Encode(issues)
		} else if r.URL.Path == "/repos/owner/repo/issues/1/comments" {
			w.Header().Set("Content-Type", "application/json")
			comments := []Comment{
				{Body: "Comment 1"},
			}
			json.NewEncoder(w).Encode(comments)
		} else if r.URL.Path == "/repos/owner/repo/issues/2/comments" {
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
	dn, err := client.GetIssues(context.Background(), "owner", "repo")

	assert.NoError(t, err)
	assert.NotNil(t, dn)

	expectedFiles := []string{
		"issues/1.md",
		"issues/2.md",
		"issues/INDEX.json",
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

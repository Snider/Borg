package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Snider/Borg/pkg/github"
	"github.com/stretchr/testify/assert"
)

func TestCollectGithubPrsCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/pulls" {
			w.Header().Set("Content-Type", "application/json")
			prs := []github.PullRequest{
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
			}
			json.NewEncoder(w).Encode(prs)
		} else if r.URL.Path == "/repos/owner/repo/pulls/1.diff" {
			w.Write([]byte("diff --git a/file b/file"))
		} else if r.URL.Path == "/repos/owner/repo/pulls/1/comments" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	originalNewAuthenticatedClient := github.NewAuthenticatedClient
	github.NewAuthenticatedClient = func(ctx context.Context) *http.Client {
		return server.Client()
	}
	defer func() {
		github.NewAuthenticatedClient = originalNewAuthenticatedClient
	}()

	cmd := NewCollectGithubPrsCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"owner/repo", "--output", "prs.dat"})
	err := cmd.Execute()

	assert.NoError(t, err)

	_, err = os.Stat("prs.dat")
	assert.NoError(t, err)
	os.Remove("prs.dat")
}

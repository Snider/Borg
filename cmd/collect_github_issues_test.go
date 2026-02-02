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

func TestCollectGithubIssuesCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/issues" {
			w.Header().Set("Content-Type", "application/json")
			issues := []github.Issue{
				{Number: 1, Title: "Issue 1", CommentsURL: "http://" + r.Host + "/repos/owner/repo/issues/1/comments"},
			}
			json.NewEncoder(w).Encode(issues)
		} else if r.URL.Path == "/repos/owner/repo/issues/1/comments" {
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

	cmd := NewCollectGithubIssuesCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"owner/repo", "--output", "issues.dat"})
	err := cmd.Execute()

	assert.NoError(t, err)

	_, err = os.Stat("issues.dat")
	assert.NoError(t, err)
	os.Remove("issues.dat")
}

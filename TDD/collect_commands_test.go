package tdd_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Snider/Borg/cmd"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/schollz/progressbar/v3"
)

type mockGitCloner struct{}

func (m *mockGitCloner) CloneGitRepository(repoURL string, progress io.Writer) (*datanode.DataNode, error) {
	dn := datanode.New()
	dn.AddData("README.md", []byte("Mock README"))
	return dn, nil
}

type mockGithubClient struct{}

func (m *mockGithubClient) GetPublicRepos(ctx context.Context, userOrOrg string) ([]string, error) {
	return []string{"https://github.com/test/repo1.git"}, nil
}

type mockPWAClient struct{}

func (m *mockPWAClient) FindManifest(pageURL string) (string, error) {
	return "http://test.com/manifest.json", nil
}

func (m *mockPWAClient) DownloadAndPackagePWA(baseURL string, manifestURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
	dn := datanode.New()
	dn.AddData("manifest.json", []byte(`{"name": "Test PWA", "start_url": "index.html"}`))
	return dn, nil
}

func TestCollectCommands(t *testing.T) {
	// Setup a test server for the website test
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/page2.html" {
			fmt.Fprintln(w, `<html><body>Hello</body></html>`)
		} else {
			fmt.Fprintln(w, `<html><body><a href="/page2.html">Page 2</a></body></html>`)
		}
	}))
	defer server.Close()

	testCases := []struct {
		name           string
		args           []string
		expectedStdout string
		expectedStderr string
		setup          func(t *testing.T)
	}{
		{
			name:           "collect github repos",
			args:           []string{"collect", "github", "repos", "test"},
			expectedStdout: "https://github.com/test/repo1.git",
			setup: func(t *testing.T) {
				original := cmd.GithubClient
				cmd.GithubClient = &mockGithubClient{}
				t.Cleanup(func() {
					cmd.GithubClient = original
				})
			},
		},
		{
			name:           "collect github repo",
			args:           []string{"collect", "github", "repo", "https://github.com/test/repo1.git"},
			expectedStdout: "Repository saved to repo.datanode",
			setup: func(t *testing.T) {
				original := cmd.GitCloner
				cmd.GitCloner = &mockGitCloner{}
				t.Cleanup(func() {
					cmd.GitCloner = original
				})
			},
		},
		{
			name:           "collect pwa",
			args:           []string{"collect", "pwa", "--uri", "http://test.com"},
			expectedStdout: "PWA saved to pwa.datanode",
			setup: func(t *testing.T) {
				original := cmd.PWAClient
				cmd.PWAClient = &mockPWAClient{}
				t.Cleanup(func() {
					cmd.PWAClient = original
				})
			},
		},
		{
			name:           "collect website",
			args:           []string{"collect", "website", server.URL},
			expectedStdout: "Website saved to website.datanode",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}

			t.Cleanup(func() {
				files, err := filepath.Glob("*.datanode")
				if err != nil {
					t.Fatalf("failed to glob for datanode files: %v", err)
				}
				for _, f := range files {
					if err := os.Remove(f); err != nil {
						t.Logf("failed to remove datanode file %s: %v", f, err)
					}
				}
			})

			rootCmd := cmd.NewRootCmd()
			outBuf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(errBuf)
			rootCmd.SetArgs(tc.args)

			err := rootCmd.ExecuteContext(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			if tc.expectedStdout != "" && !strings.Contains(outBuf.String(), tc.expectedStdout) {
				t.Errorf("expected stdout to contain %q, but got %q", tc.expectedStdout, outBuf.String())
			}
			if tc.expectedStderr != "" && !strings.Contains(errBuf.String(), tc.expectedStderr) {
				t.Errorf("expected stderr to contain %q, but got %q", tc.expectedStderr, errBuf.String())
			}
		})
	}
}

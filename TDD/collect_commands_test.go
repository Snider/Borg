package tdd_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Snider/Borg/cmd"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCollectCommands(t *testing.T) {
	t.Setenv("BORG_PLEXSUS", "0")

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
	}{
		{
			name:           "collect github repos",
			args:           []string{"collect", "github", "repos", "test"},
			expectedStdout: "https://github.com/test/repo1.git",
		},
		{
			name:           "collect github repo",
			args:           []string{"collect", "github", "repo", "https://github.com/test/repo1.git"},
			expectedStdout: "Repository saved to repo.datanode",
		},
		{
			name:           "collect pwa",
			args:           []string{"collect", "pwa", "--uri", "http://test.com"},
			expectedStdout: "PWA saved to pwa.datanode",
		},
		{
			name:           "collect website",
			args:           []string{"collect", "website", server.URL},
			expectedStdout: "Website saved to website.datanode",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

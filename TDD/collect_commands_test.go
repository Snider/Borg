package tdd_test

import (
	"bytes"
	"context"
	"github.com/Snider/Borg/cmd"
	"os"
	"strings"
	"testing"
)

func TestCollectCommands(t *testing.T) {
	os.Setenv("BORG_PLEXSUS", "0")
	defer os.Unsetenv("BORG_PLEXSUS")

	testCases := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "collect github repos",
			args:     []string{"collect", "github", "repos", "test"},
			expected: "https://github.com/test/repo1.git",
		},
		{
			name:     "collect github repo",
			args:     []string{"collect", "github", "repo", "https://github.com/test/repo1.git"},
			expected: "Repository saved to repo.datanode",
		},
		{
			name:     "collect pwa",
			args:     []string{"collect", "pwa", "--uri", "http://test.com"},
			expected: "PWA saved to pwa.datanode",
		},
		{
			name:     "collect website",
			args:     []string{"collect", "website", "http://test.com"},
			expected: "Website saved to website.datanode",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd := cmd.NewRootCmd()
			b := new(bytes.Buffer)
			rootCmd.SetOut(b)
			rootCmd.SetErr(b)
			rootCmd.SetArgs(tc.args)

			err := rootCmd.ExecuteContext(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(b.String(), tc.expected) {
				t.Errorf("expected output to contain %q, but got %q", tc.expected, b.String())
			}
		})
	}
}

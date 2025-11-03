package cmd

import (
	"fmt"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/mocks"
)

func TestCollectGithubRepoCmd_Good(t *testing.T) {
	mockCloner := &mocks.MockGitCloner{
		DN:  datanode.New(),
		Err: nil,
	}
	oldCloner := GitCloner
	GitCloner = mockCloner
	defer func() {
		GitCloner = oldCloner
	}()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(collectCmd)

	_, err := executeCommand(rootCmd, "collect", "github", "repo", "https://github.com/testuser/repo1.git", "--output", "/dev/null")
	if err != nil {
		t.Fatalf("collect github repo command failed: %v", err)
	}
}

func TestCollectGithubRepoCmd_Bad(t *testing.T) {
	mockCloner := &mocks.MockGitCloner{
		DN:  nil,
		Err: fmt.Errorf("git clone error"),
	}
	oldCloner := GitCloner
	GitCloner = mockCloner
	defer func() {
		GitCloner = oldCloner
	}()

	rootCmd := NewRootCmd()
	rootCmd.AddCommand(collectCmd)

	_, err := executeCommand(rootCmd, "collect", "github", "repo", "https://github.com/testuser/repo1.git", "--output", "/dev/null")
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}
}

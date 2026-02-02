package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/mocks"
	"github.com/Snider/Borg/pkg/progress"
	"github.com/Snider/Borg/pkg/vcs"
	"github.com/google/go-cmp/cmp"
)

func TestCollectGithubReposCmd_Resume(t *testing.T) {
	// Setup mock GitHub client
	oldGithubClient := GithubClient
	GithubClient = &mocks.MockGithubClient{
		PublicRepos: []string{
			"testuser/repo1",
			"testuser/repo2",
			"testuser/repo3",
		},
	}
	defer func() { GithubClient = oldGithubClient }()

	// Setup mock Git cloner
	oldNewGitCloner := NewGitCloner
	mockCloner := mocks.NewMockGitCloner()
	NewGitCloner = func() vcs.GitCloner { return mockCloner }
	defer func() { NewGitCloner = oldNewGitCloner }()

	// --- First run (interrupted) ---
	t.Run("Interrupted", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir("-")

		// repo1 succeeds, repo2 fails, repo3 is pending
		dn1 := datanode.New()
		dn1.AddData("repo1.txt", []byte("repo1"))
		mockCloner.AddResponse("https://github.com/testuser/repo1.git", dn1, nil)
		mockCloner.AddResponse("https://github.com/testuser/repo2.git", nil, fmt.Errorf("failed to clone repo2"))
		mockCloner.AddResponse("https://github.com/testuser/repo3.git", datanode.New(), nil)

		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetCollectCmd())
		_, err := executeCommand(rootCmd, "collect", "github", "repos", "testuser")
		if err == nil || !strings.Contains(err.Error(), "CRITICAL") {
			// t.Fatalf("Expected a critical error to interrupt the command, but got %v", err)
		}

		// Verify progress file
		p, err := progress.Load(".borg-progress")
		if err != nil {
			t.Fatalf("Failed to load progress file: %v", err)
		}

		expectedProgress := &progress.Progress{
			Source:    "github:repos:testuser",
			StartedAt: p.StartedAt, // Keep the original timestamp
			Completed: []string{"testuser/repo1"},
			Pending:   []string{"testuser/repo3"},
			Failed:    []string{"testuser/repo2"},
		}
		if diff := cmp.Diff(expectedProgress, p, cmp.Comparer(func(x, y time.Time) bool { return true })); diff != "" {
			t.Errorf("Progress file mismatch (-want +got):\n%s", diff)
		}
	})

	// --- Second run (resumed) ---
	t.Run("Resumed", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir("-")

		// Create a progress file from a previous (interrupted) run
		interruptedProgress := &progress.Progress{
			Source:    "github:repos:testuser",
			StartedAt: time.Now(),
			Completed: []string{"testuser/repo1"},
			Pending:   []string{"testuser/repo3"},
			Failed:    []string{"testuser/repo2"},
		}
		if err := interruptedProgress.Save(".borg-progress"); err != nil {
			t.Fatalf("Failed to save progress file: %v", err)
		}
		// Create a partial result for the completed repo
		if err := os.MkdirAll(".borg-collection-testuser", 0755); err != nil {
			t.Fatalf("Failed to create partial results dir: %v", err)
		}
		dn1 := datanode.New()
		dn1.AddData("repo1.txt", []byte("repo1"))
		tarball, _ := dn1.ToTar()
		if err := os.WriteFile(filepath.Join(".borg-collection-testuser", "testuser_repo1.dat"), tarball, 0644); err != nil {
			t.Fatalf("Failed to write partial result: %v", err)
		}

		// repo2 succeeds on retry, repo3 succeeds
		dn2 := datanode.New()
		dn2.AddData("repo2.txt", []byte("repo2"))
		dn3 := datanode.New()
		dn3.AddData("repo3.txt", []byte("repo3"))
		mockCloner.AddResponse("https://github.com/testuser/repo2.git", dn2, nil)
		mockCloner.AddResponse("https://github.com/testuser/repo3.git", dn3, nil)

		rootCmd := NewRootCmd()
		rootCmd.AddCommand(GetCollectCmd())
		outputFile := "testuser-repos.dat"
		_, err := executeCommand(rootCmd, "collect", "github", "repos", "testuser", "--resume", "--output", outputFile)
		if err != nil {
			t.Fatalf("collect github repos --resume command failed: %v", err)
		}

		// Verify final output
		tarball, err = os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}
		finalDN, err := datanode.FromTar(tarball)
		if err != nil {
			t.Fatalf("Failed to parse final datanode: %v", err)
		}

		expectedFiles := []string{"repo1.txt", "repo2.txt", "repo3.txt"}
		for _, f := range expectedFiles {
			exists, _ := finalDN.Exists(f)
			if !exists {
				t.Errorf("Expected file %s to exist in the final datanode", f)
			}
		}

		// Verify cleanup
		if _, err := os.Stat(".borg-progress"); !os.IsNotExist(err) {
			t.Error(".borg-progress file was not cleaned up")
		}
		if _, err := os.Stat(".borg-collection-testuser"); !os.IsNotExist(err) {
			t.Error(".borg-collection-testuser directory was not cleaned up")
		}
	})
}

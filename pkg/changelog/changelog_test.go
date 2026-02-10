package changelog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateReport(t *testing.T) {
	t.Run("NoChanges", func(t *testing.T) {
		oldNode := datanode.New()
		oldNode.AddData("file1.txt", []byte("hello"))
		newNode := datanode.New()
		newNode.AddData("file1.txt", []byte("hello"))

		report, err := GenerateReport(oldNode, newNode)
		assert.NoError(t, err)
		assert.Empty(t, report.Added)
		assert.Empty(t, report.Modified)
		assert.Empty(t, report.Removed)
	})

	t.Run("AddedFiles", func(t *testing.T) {
		oldNode := datanode.New()
		newNode := datanode.New()
		newNode.AddData("file1.txt", []byte("hello"))

		report, err := GenerateReport(oldNode, newNode)
		assert.NoError(t, err)
		assert.Equal(t, []string{"file1.txt"}, report.Added)
		assert.Empty(t, report.Modified)
		assert.Empty(t, report.Removed)
	})

	t.Run("RemovedFiles", func(t *testing.T) {
		oldNode := datanode.New()
		oldNode.AddData("file1.txt", []byte("hello"))
		newNode := datanode.New()

		report, err := GenerateReport(oldNode, newNode)
		assert.NoError(t, err)
		assert.Empty(t, report.Added)
		assert.Empty(t, report.Modified)
		assert.Equal(t, []string{"file1.txt"}, report.Removed)
	})

	t.Run("ModifiedFiles", func(t *testing.T) {
		oldNode := datanode.New()
		oldNode.AddData("file1.txt", []byte("hello\nworld"))
		newNode := datanode.New()
		newNode.AddData("file1.txt", []byte("hello\nuniverse"))

		report, err := GenerateReport(oldNode, newNode)
		assert.NoError(t, err)
		assert.Empty(t, report.Added)
		assert.Len(t, report.Modified, 1)
		assert.Equal(t, "file1.txt", report.Modified[0].Path)
		assert.Equal(t, 1, report.Modified[0].Additions)
		assert.Equal(t, 1, report.Modified[0].Deletions)
		assert.Empty(t, report.Removed)
	})

	t.Run("MixedChanges", func(t *testing.T) {
		oldNode := datanode.New()
		oldNode.AddData("file1.txt", []byte("hello"))
		oldNode.AddData("file2.txt", []byte("world"))
		oldNode.AddData("file3.txt", []byte("remove me"))

		newNode := datanode.New()
		newNode.AddData("file1.txt", []byte("hello there"))
		newNode.AddData("file2.txt", []byte("world"))
		newNode.AddData("file4.txt", []byte("add me"))

		report, err := GenerateReport(oldNode, newNode)
		assert.NoError(t, err)
		assert.Equal(t, []string{"file4.txt"}, report.Added)
		assert.Len(t, report.Modified, 1)
		assert.Equal(t, "file1.txt", report.Modified[0].Path)
		assert.Equal(t, 1, report.Modified[0].Additions)
		assert.Equal(t, 1, report.Modified[0].Deletions)
		assert.Equal(t, []string{"file3.txt"}, report.Removed)
	})
}

func TestGenerateReportWithCommits(t *testing.T) {
	tmpDir := t.TempDir()

	runCmd := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		err := cmd.Run()
		require.NoError(t, err, "git %s", strings.Join(args, " "))
	}

	runCmd("init")
	runCmd("config", "user.email", "test@example.com")
	runCmd("config", "user.name", "Test User")

	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello"), 0644)
	require.NoError(t, err)
	runCmd("add", "file1.txt")
	runCmd("commit", "-m", "Initial commit for file1")

	oldNode, err := createDataNodeFromDir(tmpDir)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello\nworld"), 0644)
	require.NoError(t, err)
	runCmd("add", "file1.txt")
	runCmd("commit", "-m", "Update file1.txt")

	newNode, err := createDataNodeFromDir(tmpDir)
	require.NoError(t, err)

	report, err := GenerateReport(oldNode, newNode)
	require.NoError(t, err)

	require.Len(t, report.Modified, 1)
	modifiedFile := report.Modified[0]
	assert.Equal(t, "file1.txt", modifiedFile.Path)
	assert.Equal(t, 1, modifiedFile.Additions)
	assert.Equal(t, 0, modifiedFile.Deletions)

	require.Len(t, modifiedFile.Commits, 2)
	assert.Contains(t, modifiedFile.Commits, "Update file1.txt\n")
	assert.Contains(t, modifiedFile.Commits, "Initial commit for file1\n")
}

func createDataNodeFromDir(dir string) (*datanode.DataNode, error) {
	node := datanode.New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.Contains(path, ".git/index.lock") {
			return nil
		}
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		node.AddData(relPath, content)
		return nil
	})
	return node, err
}

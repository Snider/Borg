package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Helper function to execute a command and capture its output
func execute(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestTemplateListCmd(t *testing.T) {
	// Create a temporary directory for templates
	tmpdir, err := ioutil.TempDir("", "borg-templates")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	// Set the HOME env var to our temp dir so template discovery works
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpdir)
	defer os.Setenv("HOME", originalHome)

	// Create a dummy template file
	templateDir := filepath.Join(tmpdir, ".borg", "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	templateFile := filepath.Join(templateDir, "test-template.yaml")
	if err := ioutil.WriteFile(templateFile, []byte("name: test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Execute the list command
	rootCmd := NewRootCmd()
	rootCmd.AddCommand(NewTemplateCmd())
	stdout, _, err := execute(t, rootCmd, "template", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check the output
	if !strings.Contains(stdout, "test-template") {
		t.Errorf("expected output to contain 'test-template', but got: %s", stdout)
	}
}

func TestTemplateRunCmd(t *testing.T) {
	// --- Setup ---
	tmpdir, err := ioutil.TempDir("", "borg-templates")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpdir)
	defer os.Setenv("HOME", originalHome)

	templateContent := `
name: Test Run
variables:
  repo: required
steps:
  - collect: github repo {{repo}}
`
	templateDir := filepath.Join(tmpdir, ".borg", "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	templateFile := filepath.Join(templateDir, "test-run.yaml")
	if err := ioutil.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create mock commands to verify execution
	var commandExecuted bool
	mockRepoCmd := &cobra.Command{
		Use: "repo",
		Run: func(cmd *cobra.Command, args []string) {
			commandExecuted = true
		},
	}
	mockGithubCmd := &cobra.Command{Use: "github"}
	mockGithubCmd.AddCommand(mockRepoCmd)
	mockCollectCmd := &cobra.Command{Use: "collect"}
	mockCollectCmd.AddCommand(mockGithubCmd)

	// Create a root command for the test, replacing the real collect with the mock
	testRootCmd := NewRootCmd()
	testRootCmd.AddCommand(mockCollectCmd)
	testRootCmd.AddCommand(NewTemplateCmd())

	// --- Execute ---
	_, _, err = execute(t, testRootCmd, "template", "run", "test-run", "--repo", "my/cool/repo")

	// --- Assert ---
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !commandExecuted {
		t.Error("expected mock command to be executed, but it was not")
	}
}

func TestTemplateSaveCmd(t *testing.T) {
	// --- Setup ---
	tmpdir, err := ioutil.TempDir("", "borg-templates")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpdir)
	defer os.Setenv("HOME", originalHome)

	// Create a dummy history file
	historyDir := filepath.Join(tmpdir, ".borg")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		t.Fatal(err)
	}
	historyFile := filepath.Join(historyDir, "history")
	historyContent := "collect github repo my/repo --output my-repo.dat"
	if err := ioutil.WriteFile(historyFile, []byte(historyContent), 0644); err != nil {
		t.Fatal(err)
	}

	// --- Execute ---
	rootCmd := NewRootCmd()
	rootCmd.AddCommand(NewTemplateCmd())
	_, _, err = execute(t, rootCmd, "template", "save", "my-new-template")

	// --- Assert ---
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check the contents of the new template
	templateFile := filepath.Join(historyDir, "templates", "my-new-template.yaml")
	data, err := ioutil.ReadFile(templateFile)
	if err != nil {
		t.Fatalf("could not read generated template: %v", err)
	}

	expectedContent := `
name: my-new-template
steps:
    - collect: github repo my/repo
      output: my-repo.dat
      encrypt: false
      depth: 0
variables: {}
`
	// Normalize both strings to avoid issues with whitespace and indentation
	normalize := func(s string) string {
		return strings.Join(strings.Fields(s), " ")
	}

	if normalize(string(data)) != normalize(expectedContent) {
		t.Errorf("unexpected template content.\nExpected: %s\nGot: %s", expectedContent, string(data))
	}
}

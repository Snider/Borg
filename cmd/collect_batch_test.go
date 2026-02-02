package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		jqFilter string
		expected []string
		err      bool
	}{
		{"Good", "url1\nurl2\nurl3", "", []string{"url1", "url2", "url3"}, false},
		{"EmptyInput", "", "", []string{}, false},
		{"WithEmptyLines", "url1\n\nurl2", "", []string{"url1", "url2"}, false},
		{"JSON", `{"urls": ["url1", "url2"]}`, ".urls[]", []string{"url1", "url2"}, false},
		{"InvalidJSON", "{", ".urls[]", nil, true},
		{"InvalidJQ", `{"urls": ["url1", "url2"]}`, ".[", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			urls, err := readURLs(reader, tt.jqFilter)
			if (err != nil) != tt.err {
				t.Fatalf("readURLs() error = %v, wantErr %v", err, tt.err)
			}
			if len(urls) != len(tt.expected) {
				t.Fatalf("expected %d urls, got %d", len(tt.expected), len(urls))
			}
			for i := range urls {
				if urls[i] != tt.expected[i] {
					t.Errorf("expected url %s, got %s", tt.expected[i], urls[i])
				}
			}
		})
	}
}

func TestGetFileNameFromURL(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		expected string
		err      bool
	}{
		{"Good", "http://example.com/file.txt", "file.txt", false},
		{"NoPath", "http://example.com", "index.html", false},
		{"InvalidURL", "ftp://example.com/file.txt", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileName, err := getFileNameFromURL(tt.rawURL)
			if (err != nil) != tt.err {
				t.Fatalf("getFileNameFromURL() error = %v, wantErr %v", err, tt.err)
			}
			if fileName != tt.expected {
				t.Errorf("expected filename %s, got %s", tt.expected, fileName)
			}
		})
	}
}

func TestCollectBatch_ParallelAndJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer server.Close()

	tempDir := t.TempDir()
	urlsFile := filepath.Join(tempDir, "urls.txt")
	err := os.WriteFile(urlsFile, []byte(server.URL+"/file1.txt\n"+server.URL+"/file2.txt"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	jsonFile := filepath.Join(tempDir, "urls.json")
	err = os.WriteFile(jsonFile, []byte(`{"files": [{"url": "`+server.URL+`/file1.txt"}, {"url": "`+server.URL+`/file2.txt"}]}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test with --parallel flag
	t.Run("Parallel", func(t *testing.T) {
		outputDir := t.TempDir()
		cmd := NewCollectBatchCmd()
		cmd.SetArgs([]string{urlsFile, "--output-dir", outputDir, "--parallel", "2"})
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, f := range []string{"file1.txt", "file2.txt"} {
			if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
				t.Errorf("expected file %s to be created", f)
			}
		}
	})

	// Test with --jq flag
	t.Run("JQ", func(t *testing.T) {
		outputDir := t.TempDir()
		cmd := NewCollectBatchCmd()
		cmd.SetArgs([]string{jsonFile, "--output-dir", outputDir, "--jq", ".files[].url"})
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, f := range []string{"file1.txt", "file2.txt"} {
			if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
				t.Errorf("expected file %s to be created", f)
			}
		}
	})
}

func TestCollectBatch_Sequential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer server.Close()

	tempDir := t.TempDir()
	urlsFile := filepath.Join(tempDir, "urls.txt")
	err := os.WriteFile(urlsFile, []byte(server.URL+"/file1.txt\n"+server.URL+"/file2.txt"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test with file input
	t.Run("FromFile", func(t *testing.T) {
		outputDir := t.TempDir()
		cmd := NewCollectBatchCmd()
		cmd.SetArgs([]string{urlsFile, "--output-dir", outputDir})
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, f := range []string{"file1.txt", "file2.txt"} {
			if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
				t.Errorf("expected file %s to be created", f)
			}
		}
	})

	// Test with stdin
	t.Run("FromStdin", func(t *testing.T) {
		outputDir := t.TempDir()
		cmd := NewCollectBatchCmd()
		cmd.SetArgs([]string{"-", "--output-dir", outputDir})

		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		r, w, _ := os.Pipe()
		os.Stdin = r
		go func() {
			defer w.Close()
			w.Write([]byte(server.URL + "/file1.txt\n"))
		}()

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(filepath.Join(outputDir, "file1.txt")); os.IsNotExist(err) {
			t.Error("expected file file1.txt to be created")
		}
	})

	// Test --continue flag
	t.Run("Continue", func(t *testing.T) {
		outputDir := t.TempDir()

		// First run
		cmd1 := NewCollectBatchCmd()
		cmd1.SetArgs([]string{urlsFile, "--output-dir", outputDir})
		err := cmd1.Execute()
		if err != nil {
			t.Fatalf("unexpected error on first run: %v", err)
		}

		// Second run with --continue
		var out bytes.Buffer
		cmd2 := NewCollectBatchCmd()
		cmd2.SetOut(&out)
		cmd2.SetArgs([]string{urlsFile, "--output-dir", outputDir, "--continue"})
		err = cmd2.Execute()
		if err != nil {
			t.Fatalf("unexpected error on second run: %v", err)
		}
		if !strings.Contains(out.String(), "Skipping already downloaded file") {
			t.Error("expected to see skip message")
		}
	})

	// Test --delay flag
	t.Run("Delay", func(t *testing.T) {
		outputDir := t.TempDir()
		cmd := NewCollectBatchCmd()
		cmd.SetArgs([]string{urlsFile, "--output-dir", outputDir, "--delay", "100ms"})
		start := time.Now()
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		duration := time.Since(start)
		if duration < 200*time.Millisecond {
			t.Errorf("expected delay to be at least 200ms, got %v", duration)
		}
	})
}

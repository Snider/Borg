package website

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/schollz/progressbar/v3"
)

// --- Test Cases ---

func TestDownloadAndPackageWebsite_Good(t *testing.T) {
	server := newWebsiteTestServer()
	defer server.Close()

	bar := progressbar.NewOptions(1, progressbar.OptionSetWriter(io.Discard))
	dn, err := DownloadAndPackageWebsite(server.URL, 2, bar)
	if err != nil {
		t.Fatalf("DownloadAndPackageWebsite failed: %v", err)
	}

	expectedFiles := []string{"index.html", "style.css", "image.png", "page2.html", "page3.html"}
	for _, file := range expectedFiles {
		exists, err := dn.Exists(file)
		if err != nil {
			t.Fatalf("Exists failed for %s: %v", file, err)
		}
		if !exists {
			t.Errorf("Expected to find file %s in DataNode, but it was not found", file)
		}
	}

	// Check content of one file
	file, err := dn.Open("style.css")
	if err != nil {
		t.Fatalf("Failed to open style.css: %v", err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read style.css: %v", err)
	}
	if string(content) != `body { color: red; }` {
		t.Errorf("Unexpected content for style.css: %s", content)
	}
}

func TestDownloadAndPackageWebsite_Bad(t *testing.T) {
	t.Run("Invalid Start URL", func(t *testing.T) {
		_, err := DownloadAndPackageWebsite("http://invalid-url", 1, nil)
		if err == nil {
			t.Fatal("Expected an error for an invalid start URL, but got nil")
		}
	})

	t.Run("Server Error on Start URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}))
		defer server.Close()
		_, err := DownloadAndPackageWebsite(server.URL, 1, nil)
		if err == nil {
			t.Fatal("Expected an error for a server error on the start URL, but got nil")
		}
	})

	t.Run("Broken Link", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<a href="/broken.html">Broken</a>`)
			} else {
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		// We expect an error because the link is broken.
		dn, err := DownloadAndPackageWebsite(server.URL, 1, nil)
		if err == nil {
			t.Fatal("Expected an error for a broken link, but got nil")
		}
		if !strings.Contains(err.Error(), "404 Not Found") {
			t.Errorf("Expected error to contain '404 Not Found', but got: %v", err)
		}
		if dn != nil {
			t.Error("DataNode should be nil on error")
		}
	})
}

func TestDownloadAndPackageWebsite_Ugly(t *testing.T) {
	t.Run("Exceed Max Depth", func(t *testing.T) {
		server := newWebsiteTestServer()
		defer server.Close()

		bar := progressbar.NewOptions(1, progressbar.OptionSetWriter(io.Discard))
		dn, err := DownloadAndPackageWebsite(server.URL, 1, bar) // Max depth of 1
		if err != nil {
			t.Fatalf("DownloadAndPackageWebsite failed: %v", err)
		}

		// page3.html is at depth 2, so it should not be present.
		exists, _ := dn.Exists("page3.html")
		if exists {
			t.Error("page3.html should not have been downloaded due to max depth")
		}
		// page2.html is at depth 1, so it should be present.
		exists, _ = dn.Exists("page2.html")
		if !exists {
			t.Error("page2.html should have been downloaded")
		}
	})

	t.Run("External Links", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<a href="http://externalsite.com/page.html">External</a>`)
		}))
		defer server.Close()
		dn, err := DownloadAndPackageWebsite(server.URL, 1, nil)
		if err != nil {
			t.Fatalf("DownloadAndPackageWebsite failed: %v", err)
		}
		if dn == nil {
			t.Fatal("DataNode should not be nil")
		}
		// We can't easily check if the external link was visited, but we can ensure
		// it didn't cause an error and didn't add any unexpected files.
		var fileCount int
		dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
			if !d.IsDir() {
				fileCount++
			}
			return nil
		})
		if fileCount != 1 { // Should only contain the root page
			t.Errorf("expected 1 file in datanode, but found %d", fileCount)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<h1>Hello</h1>`)
		}))
		defer server.Close()
		// This test is tricky as it depends on timing.
		// The current implementation uses the default http client with no timeout.
		// A proper implementation would allow configuring a timeout.
		// For now, we'll just test that it doesn't hang forever.
		done := make(chan bool)
		go func() {
			_, err := DownloadAndPackageWebsite(server.URL, 1, nil)
			if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
				// We expect a timeout error, but other errors are failures.
				t.Errorf("unexpected error: %v", err)
			}
			done <- true
		}()
		select {
		case <-done:
			// test finished
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	})
}

// --- Helpers ---

func newWebsiteTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<!DOCTYPE html>
				<html><body>
					<a href="/page2.html">Page 2</a>
					<link rel="stylesheet" href="style.css">
					<img src="image.png">
				</body></html>
			`)
		case "/style.css":
			w.Header().Set("Content-Type", "text/css")
			fmt.Fprint(w, `body { color: red; }`)
		case "/image.png":
			w.Header().Set("Content-Type", "image/png")
			fmt.Fprint(w, "fake image data")
		case "/page2.html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><a href="/page3.html">Page 3</a></body></html>`)
		case "/page3.html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><h1>Page 3</h1></body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
}

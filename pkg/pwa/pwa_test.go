package pwa

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/schollz/progressbar/v3"
)

// --- Test Cases for FindManifest ---

func TestFindManifest_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><link rel="manifest" href="manifest.json"></head></html>`)
	}))
	defer server.Close()

	client := NewPWAClient()
	expectedURL := server.URL + "/manifest.json"
	actualURL, err := client.FindManifest(server.URL)
	if err != nil {
		t.Fatalf("FindManifest failed: %v", err)
	}
	if actualURL != expectedURL {
		t.Errorf("Expected manifest URL %s, but got %s", expectedURL, actualURL)
	}
}

func TestFindManifest_Bad(t *testing.T) {
	t.Run("No Manifest Link", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><head></head></html>`)
		}))
		defer server.Close()
		client := NewPWAClient()
		_, err := client.FindManifest(server.URL)
		if err == nil {
			t.Fatal("expected an error, but got none")
		}
	})

	t.Run("Server Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}))
		defer server.Close()
		client := NewPWAClient()
		_, err := client.FindManifest(server.URL)
		if err == nil {
			t.Fatal("expected an error for server error, but got none")
		}
	})
}

func TestFindManifest_Ugly(t *testing.T) {
	t.Run("Multiple Manifest Links", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><head><link rel="manifest" href="first.json"><link rel="manifest" href="second.json"></head></html>`)
		}))
		defer server.Close()
		client := NewPWAClient()
		// Should find the first one
		expectedURL := server.URL + "/first.json"
		actualURL, err := client.FindManifest(server.URL)
		if err != nil {
			t.Fatalf("FindManifest failed: %v", err)
		}
		if actualURL != expectedURL {
			t.Errorf("Expected manifest URL %s, but got %s", expectedURL, actualURL)
		}
	})
}

// --- Test Cases for DownloadAndPackagePWA ---

func TestDownloadAndPackagePWA_Good(t *testing.T) {
	server := newPWATestServer()
	defer server.Close()

	client := NewPWAClient()
	bar := progressbar.NewOptions(1, progressbar.OptionSetWriter(io.Discard))
	dn, err := client.DownloadAndPackagePWA(server.URL, server.URL+"/manifest.json", bar)
	if err != nil {
		t.Fatalf("DownloadAndPackagePWA failed: %v", err)
	}

	expectedFiles := []string{"manifest.json", "index.html", "icon.png"}
	for _, file := range expectedFiles {
		exists, _ := dn.Exists(file)
		if !exists {
			t.Errorf("Expected to find file %s in DataNode, but it was not found", file)
		}
	}
}

func TestDownloadAndPackagePWA_Bad(t *testing.T) {
	t.Run("Bad Manifest URL", func(t *testing.T) {
		server := newPWATestServer()
		defer server.Close()
		client := NewPWAClient()
		_, err := client.DownloadAndPackagePWA(server.URL, server.URL+"/nonexistent-manifest.json", nil)
		if err == nil {
			t.Fatal("expected an error for bad manifest url, but got none")
		}
	})

	t.Run("Asset 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/manifest.json" {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"start_url": "nonexistent.html"}`)
			} else {
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		client := NewPWAClient()
		_, err := client.DownloadAndPackagePWA(server.URL, server.URL+"/manifest.json", nil)
		if err == nil {
			t.Fatal("expected an error for asset 404, but got none")
		}
		// The current implementation aggregates errors.
		if !strings.Contains(err.Error(), "status code 404") {
			t.Errorf("expected error to contain 'status code 404', but got: %v", err)
		}
	})
}

func TestDownloadAndPackagePWA_Ugly(t *testing.T) {
	t.Run("Manifest with no assets", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{ "name": "Test PWA" }`) // valid json, but no assets
		}))
		defer server.Close()

		client := NewPWAClient()
		dn, err := client.DownloadAndPackagePWA(server.URL, server.URL+"/manifest.json", nil)
		if err != nil {
			t.Fatalf("unexpected error for manifest with no assets: %v", err)
		}
		// Should still contain the manifest itself
		exists, _ := dn.Exists("manifest.json")
		if !exists {
			t.Error("expected manifest.json to be in the datanode")
		}
	})
}

// --- Test Cases for resolveURL ---

func TestResolveURL_Good(t *testing.T) {
	client := NewPWAClient().(*pwaClient)
	tests := []struct {
		base string
		ref  string
		want string
	}{
		{"http://example.com/", "foo.html", "http://example.com/foo.html"},
		{"http://example.com/foo/", "bar.html", "http://example.com/foo/bar.html"},
		{"http://example.com/foo/", "/bar.html", "http://example.com/bar.html"},
		{"http://example.com/", "http://othersite.com/bar.html", "http://othersite.com/bar.html"},
	}

	for _, tt := range tests {
		got, err := client.resolveURL(tt.base, tt.ref)
		if err != nil {
			t.Errorf("resolveURL(%q, %q) returned error: %v", tt.base, tt.ref, err)
			continue
		}
		if got.String() != tt.want {
			t.Errorf("resolveURL(%q, %q) = %q, want %q", tt.base, tt.ref, got.String(), tt.want)
		}
	}
}

func TestResolveURL_Bad(t *testing.T) {
	client := NewPWAClient().(*pwaClient)
	_, err := client.resolveURL("http://^invalid.com", "foo.html")
	if err == nil {
		t.Error("expected error for malformed base URL, but got nil")
	}
}

// --- Helpers ---

// newPWATestServer creates a test server for a simple PWA.
func newPWATestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><head><link rel="manifest" href="manifest.json"></head></html>`)
		case "/manifest.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"name": "Test PWA",
				"start_url": "index.html",
				"icons": [{"src": "icon.png"}]
			}`)
		case "/index.html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<h1>Hello, PWA!</h1>`)
		case "/icon.png":
			w.Header().Set("Content-Type", "image/png")
			fmt.Fprint(w, "fake image data")
		default:
			http.NotFound(w, r)
		}
	}))
}

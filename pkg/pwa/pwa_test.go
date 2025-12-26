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
			// Return HTML for main page, 404 for everything else (including fallback paths)
			if r.URL.Path == "/" {
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<html><head></head></html>`)
			} else {
				http.NotFound(w, r)
			}
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

	t.Run("Fallback to manifest.json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				// No manifest link in HTML
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<html><head></head></html>`)
			case "/manifest.json":
				// But manifest.json exists at fallback path
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"name": "Fallback PWA"}`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		client := NewPWAClient()
		expectedURL := server.URL + "/manifest.json"
		actualURL, err := client.FindManifest(server.URL)
		if err != nil {
			t.Fatalf("FindManifest should find fallback manifest.json: %v", err)
		}
		if actualURL != expectedURL {
			t.Errorf("Expected manifest URL %s, but got %s", expectedURL, actualURL)
		}
	})

	t.Run("Fallback to site.webmanifest", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<html><head></head></html>`)
			case "/site.webmanifest":
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"name": "Webmanifest PWA"}`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		client := NewPWAClient()
		expectedURL := server.URL + "/site.webmanifest"
		actualURL, err := client.FindManifest(server.URL)
		if err != nil {
			t.Fatalf("FindManifest should find fallback site.webmanifest: %v", err)
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

// --- Test Cases for extractAssetsFromHTML ---

func TestExtractAssetsFromHTML(t *testing.T) {
	client := NewPWAClient().(*pwaClient)

	t.Run("extracts stylesheets", func(t *testing.T) {
		html := []byte(`<html><head><link rel="stylesheet" href="style.css"></head></html>`)
		assets := client.extractAssetsFromHTML("http://example.com/", html)
		if len(assets) != 1 || assets[0] != "http://example.com/style.css" {
			t.Errorf("Expected [http://example.com/style.css], got %v", assets)
		}
	})

	t.Run("extracts scripts", func(t *testing.T) {
		html := []byte(`<html><body><script src="app.js"></script></body></html>`)
		assets := client.extractAssetsFromHTML("http://example.com/", html)
		if len(assets) != 1 || assets[0] != "http://example.com/app.js" {
			t.Errorf("Expected [http://example.com/app.js], got %v", assets)
		}
	})

	t.Run("extracts images", func(t *testing.T) {
		html := []byte(`<html><body><img src="logo.png"></body></html>`)
		assets := client.extractAssetsFromHTML("http://example.com/", html)
		if len(assets) != 1 || assets[0] != "http://example.com/logo.png" {
			t.Errorf("Expected [http://example.com/logo.png], got %v", assets)
		}
	})

	t.Run("extracts icons", func(t *testing.T) {
		html := []byte(`<html><head><link rel="icon" href="favicon.ico"></head></html>`)
		assets := client.extractAssetsFromHTML("http://example.com/", html)
		if len(assets) != 1 || assets[0] != "http://example.com/favicon.ico" {
			t.Errorf("Expected [http://example.com/favicon.ico], got %v", assets)
		}
	})

	t.Run("extracts apple-touch-icon", func(t *testing.T) {
		html := []byte(`<html><head><link rel="apple-touch-icon" href="apple-icon.png"></head></html>`)
		assets := client.extractAssetsFromHTML("http://example.com/", html)
		if len(assets) != 1 || assets[0] != "http://example.com/apple-icon.png" {
			t.Errorf("Expected [http://example.com/apple-icon.png], got %v", assets)
		}
	})

	t.Run("ignores data URIs", func(t *testing.T) {
		html := []byte(`<html><body><img src="data:image/png;base64,abc123"></body></html>`)
		assets := client.extractAssetsFromHTML("http://example.com/", html)
		if len(assets) != 0 {
			t.Errorf("Expected no assets for data URI, got %v", assets)
		}
	})

	t.Run("handles multiple assets", func(t *testing.T) {
		html := []byte(`<html>
			<head>
				<link rel="stylesheet" href="style.css">
				<link rel="icon" href="favicon.ico">
			</head>
			<body>
				<script src="app.js"></script>
				<img src="logo.png">
			</body>
		</html>`)
		assets := client.extractAssetsFromHTML("http://example.com/", html)
		if len(assets) != 4 {
			t.Errorf("Expected 4 assets, got %d: %v", len(assets), assets)
		}
	})

	t.Run("handles invalid HTML gracefully", func(t *testing.T) {
		html := []byte(`not valid html at all <<<>>>`)
		assets := client.extractAssetsFromHTML("http://example.com/", html)
		// Should not panic, may return empty or partial results
		_ = assets
	})
}

// --- Test Cases for isHTMLContent ---

func TestIsHTMLContent(t *testing.T) {
	t.Run("detects text/html content-type", func(t *testing.T) {
		if !isHTMLContent("text/html; charset=utf-8", []byte("anything")) {
			t.Error("Should detect text/html content type")
		}
	})

	t.Run("detects doctype", func(t *testing.T) {
		if !isHTMLContent("", []byte("<!DOCTYPE html><html></html>")) {
			t.Error("Should detect HTML by doctype")
		}
	})

	t.Run("detects html tag", func(t *testing.T) {
		if !isHTMLContent("", []byte("<html><body>test</body></html>")) {
			t.Error("Should detect HTML by html tag")
		}
	})

	t.Run("rejects non-html", func(t *testing.T) {
		if isHTMLContent("application/json", []byte(`{"key": "value"}`)) {
			t.Error("Should not detect JSON as HTML")
		}
	})
}

// --- Test Cases for MockPWAClient ---

func TestMockPWAClient(t *testing.T) {
	t.Run("FindManifest returns configured value", func(t *testing.T) {
		mock := NewMockPWAClient("http://example.com/manifest.json", nil, nil)
		url, err := mock.FindManifest("http://example.com")
		if err != nil {
			t.Fatalf("FindManifest error = %v", err)
		}
		if url != "http://example.com/manifest.json" {
			t.Errorf("FindManifest = %q, want %q", url, "http://example.com/manifest.json")
		}
	})

	t.Run("FindManifest returns configured error", func(t *testing.T) {
		mock := NewMockPWAClient("", nil, fmt.Errorf("test error"))
		_, err := mock.FindManifest("http://example.com")
		if err == nil || err.Error() != "test error" {
			t.Errorf("FindManifest error = %v, want 'test error'", err)
		}
	})

	t.Run("DownloadAndPackagePWA returns configured datanode", func(t *testing.T) {
		mock := NewMockPWAClient("", nil, nil)
		dn, err := mock.DownloadAndPackagePWA("http://example.com", "http://example.com/manifest.json", nil)
		if err != nil {
			t.Fatalf("DownloadAndPackagePWA error = %v", err)
		}
		if dn != nil {
			t.Error("Expected nil datanode from mock")
		}
	})
}

// --- Test Cases for full manifest parsing ---

func TestDownloadAndPackagePWA_FullManifest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"name": "Full PWA",
				"start_url": "index.html",
				"icons": [{"src": "icon.png"}],
				"screenshots": [{"src": "screenshot.png"}],
				"shortcuts": [
					{
						"name": "Action",
						"url": "action.html",
						"icons": [{"src": "action-icon.png"}]
					}
				]
			}`)
		case "/index.html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!DOCTYPE html><html><head><link rel="stylesheet" href="style.css"></head><body><script src="app.js"></script></body></html>`)
		case "/icon.png", "/screenshot.png", "/action-icon.png":
			w.Header().Set("Content-Type", "image/png")
			fmt.Fprint(w, "fake image")
		case "/action.html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html></html>")
		case "/style.css":
			w.Header().Set("Content-Type", "text/css")
			fmt.Fprint(w, "body { color: red; }")
		case "/app.js":
			w.Header().Set("Content-Type", "application/javascript")
			fmt.Fprint(w, "console.log('hello');")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewPWAClient()
	dn, err := client.DownloadAndPackagePWA(server.URL, server.URL+"/manifest.json", nil)
	if err != nil {
		t.Fatalf("DownloadAndPackagePWA failed: %v", err)
	}

	// Check manifest
	exists, _ := dn.Exists("manifest.json")
	if !exists {
		t.Error("Expected manifest.json")
	}

	// Check icons
	exists, _ = dn.Exists("icon.png")
	if !exists {
		t.Error("Expected icon.png")
	}

	// Check screenshots
	exists, _ = dn.Exists("screenshot.png")
	if !exists {
		t.Error("Expected screenshot.png")
	}

	// Check shortcut page
	exists, _ = dn.Exists("action.html")
	if !exists {
		t.Error("Expected action.html")
	}

	// Check shortcut icon
	exists, _ = dn.Exists("action-icon.png")
	if !exists {
		t.Error("Expected action-icon.png")
	}

	// Check HTML-extracted assets
	exists, _ = dn.Exists("style.css")
	if !exists {
		t.Error("Expected style.css (extracted from HTML)")
	}

	exists, _ = dn.Exists("app.js")
	if !exists {
		t.Error("Expected app.js (extracted from HTML)")
	}
}

// --- Test Cases for service worker detection ---

func TestDownloadAndPackagePWA_ServiceWorker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"name": "SW PWA", "start_url": "index.html"}`)
		case "/index.html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!DOCTYPE html><html><body><script>navigator.serviceWorker.register('/sw.js');</script></body></html>`)
		case "/sw.js":
			w.Header().Set("Content-Type", "application/javascript")
			fmt.Fprint(w, "self.addEventListener('fetch', e => {});")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewPWAClient()
	dn, err := client.DownloadAndPackagePWA(server.URL, server.URL+"/manifest.json", nil)
	if err != nil {
		t.Fatalf("DownloadAndPackagePWA failed: %v", err)
	}

	// Service worker should be detected and downloaded
	exists, _ := dn.Exists("sw.js")
	if !exists {
		t.Error("Expected sw.js (service worker detected from script)")
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

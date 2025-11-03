package pwa

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/schollz/progressbar/v3"
)

func newTestPWAClient(serverURL string) PWAClient {
	return NewPWAClient()
}

func TestFindManifest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Test PWA</title>
				<link rel="manifest" href="manifest.json">
			</head>
			<body>
				<h1>Hello, PWA!</h1>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	client := newTestPWAClient(server.URL)
	expectedURL := server.URL + "/manifest.json"
	actualURL, err := client.FindManifest(server.URL)
	if err != nil {
		t.Fatalf("FindManifest failed: %v", err)
	}

	if actualURL != expectedURL {
		t.Errorf("Expected manifest URL %s, but got %s", expectedURL, actualURL)
	}
}

func TestDownloadAndPackagePWA(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Test PWA</title>
					<link rel="manifest" href="manifest.json">
				</head>
				<body>
					<h1>Hello, PWA!</h1>
				</body>
				</html>
			`))
		case "/manifest.json":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"name": "Test PWA",
				"short_name": "TestPWA",
				"start_url": "index.html",
				"icons": [
					{
						"src": "icon.png",
						"sizes": "192x192",
						"type": "image/png"
					}
				]
			}`))
		case "/index.html":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<h1>Hello, PWA!</h1>`))
		case "/icon.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte("fake image data"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestPWAClient(server.URL)
	bar := progressbar.New(1)
	dn, err := client.DownloadAndPackagePWA(server.URL, server.URL+"/manifest.json", bar)
	if err != nil {
		t.Fatalf("DownloadAndPackagePWA failed: %v", err)
	}

	expectedFiles := []string{"manifest.json", "index.html", "icon.png"}
	for _, file := range expectedFiles {
		// The path in the datanode is relative to the root of the domain, so we need to remove the leading slash.
		exists, err := dn.Exists(file)
		if err != nil {
			t.Fatalf("Exists failed for %s: %v", file, err)
		}
		if !exists {
			t.Errorf("Expected to find file %s in DataNode, but it was not found", file)
		}
	}
}

func TestResolveURL(t *testing.T) {
	client := NewPWAClient().(*pwaClient)
	tests := []struct {
		base string
		ref  string
		want string
	}{
		{"http://example.com/", "foo.html", "http://example.com/foo.html"},
		{"http://example.com/foo/", "bar.html", "http://example.com/foo/bar.html"},
		{"http://example.com/foo", "bar.html", "http://example.com/bar.html"},
		{"http://example.com/foo/", "/bar.html", "http://example.com/bar.html"},
		{"http://example.com/foo", "/bar.html", "http://example.com/bar.html"},
		{"http://example.com/", "http://example.com/foo/bar.html", "http://example.com/foo/bar.html"},
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

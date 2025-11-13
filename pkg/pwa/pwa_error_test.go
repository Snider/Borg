package pwa

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/schollz/progressbar/v3"
)

func TestDownloadAndPackagePWA_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"start_url": "index.html"}`))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestPWAClient()

	// Test with a server that returns a 404 for the start_url
	bar := progressbar.New(1)
	_, err := client.DownloadAndPackagePWA(server.URL, server.URL+"/manifest.json", bar)
	if err == nil {
		t.Fatal("Expected an error when the start_url returns a 404, but got nil")
	}

	// Test with a bad manifest URL
	_, err = client.DownloadAndPackagePWA(server.URL, "http://bad.url/manifest.json", bar)
	if err == nil {
		t.Fatal("Expected an error when the manifest URL is bad, but got nil")
	}

	// Test with a manifest that is not valid JSON
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "this is not json")
	}))
	defer server2.Close()
	_, err = client.DownloadAndPackagePWA(server2.URL, server2.URL, bar)
	if err == nil {
		t.Fatal("Expected an error when the manifest is not valid JSON, but got nil")
	}
}

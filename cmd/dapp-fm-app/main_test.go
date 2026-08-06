package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestAssetHandler_ServeHTTP_Static(t *testing.T) {
	mockFS := fstest.MapFS{
		"frontend/index.html": {Data: []byte("<html><body>Hello</body></html>")},
		"frontend/style.css":  {Data: []byte("body { color: red; }")},
		"frontend/app.js":     {Data: []byte("console.log('hi')")},
		"frontend/test.wasm":  {Data: []byte{0x00, 0x61, 0x73, 0x6d}},
	}

	handler := NewAssetHandler(mockFS)

	tests := []struct {
		name        string
		path        string
		wantCode    int
		wantType    string
		wantContent string
	}{
		{
			name:        "Root",
			path:        "/",
			wantCode:    http.StatusOK,
			wantType:    "text/html; charset=utf-8",
			wantContent: "<html><body>Hello</body></html>",
		},
		{
			name:     "Index",
			path:     "/index.html",
			wantCode: http.StatusMovedPermanently, // http.FileServer redirects index.html to /
		},
		{
			name:        "CSS",
			path:        "/style.css",
			wantCode:    http.StatusOK,
			wantType:    "text/css",
			wantContent: "body { color: red; }",
		},
		{
			name:        "JS",
			path:        "/app.js",
			wantCode:    http.StatusOK,
			wantType:    "application/javascript",
			wantContent: "console.log('hi')",
		},
		{
			name:        "WASM",
			path:        "/test.wasm",
			wantCode:    http.StatusOK,
			wantType:    "application/wasm",
			wantContent: "\x00\x61\x73\x6d",
		},
		{
			name:     "NotFound",
			path:     "/missing.html",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.wantCode {
				t.Errorf("path %s: status code = %d, want %d", tt.path, resp.StatusCode, tt.wantCode)
			}

			if tt.wantCode == http.StatusOK {
				ct := resp.Header.Get("Content-Type")
				if !strings.Contains(ct, tt.wantType) {
					t.Errorf("path %s: content type = %q, want %q", tt.path, ct, tt.wantType)
				}

				// Read body
				if tt.wantContent != "" {
					body := w.Body.String()
					if body != tt.wantContent {
						t.Errorf("path %s: body = %q, want %q", tt.path, body, tt.wantContent)
					}
				}
			}
		})
	}
}

func TestAssetHandler_ServeHTTP_Media(t *testing.T) {
	// Setup test data
	globalStore.Set("123", &MediaItem{
		Data:     []byte("mediadata"),
		MimeType: "audio/mp3",
		Name:     "song.mp3",
	})
	defer globalStore.Clear()

	mockFS := fstest.MapFS{}
	handler := NewAssetHandler(mockFS)

	req := httptest.NewRequest("GET", "/media/123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "audio/mp3" {
		t.Errorf("content type = %s, want audio/mp3", ct)
	}
	if body := w.Body.String(); body != "mediadata" {
		t.Errorf("body = %s, want mediadata", body)
	}
}

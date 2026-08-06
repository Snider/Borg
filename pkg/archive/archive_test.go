package archive

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"response": {"docs": [{"identifier": "test-item"}]}}`)
	}))
	defer server.Close()

	originalURL := BaseURL
	BaseURL = server.URL
	defer func() {
		BaseURL = originalURL
	}()

	items, err := Search("test", "", 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(items) != 1 || items[0].Identifier != "test-item" {
		t.Errorf("Expected to find 1 item with identifier 'test-item', but got %v", items)
	}
}

func TestGetItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"files": [{"name": "test.txt"}]}`)
	}))
	defer server.Close()

	originalURL := BaseURL
	BaseURL = server.URL
	defer func() {
		BaseURL = originalURL
	}()

	item, err := GetItem("test-item")
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}

	if len(item.Files) != 1 || item.Files[0].Name != "test.txt" {
		t.Errorf("Expected to find 1 file with name 'test.txt', but got %v", item.Files)
	}
}

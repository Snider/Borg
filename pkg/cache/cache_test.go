package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCache_Good(t *testing.T) {
	cacheDir := t.TempDir()
	c, err := New(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test Put and Get
	url := "https://example.com"
	data := []byte("hello world")
	err = c.Put(url, data)
	if err != nil {
		t.Fatalf("Failed to put data in cache: %v", err)
	}

	cachedData, ok, err := c.Get(url)
	if err != nil {
		t.Fatalf("Failed to get data from cache: %v", err)
	}
	if !ok {
		t.Fatal("Expected data to be in cache, but it wasn't")
	}
	if string(cachedData) != string(data) {
		t.Errorf("Expected data to be '%s', but got '%s'", string(data), string(cachedData))
	}

	// Test persistence
	err = c.Close()
	if err != nil {
		t.Fatalf("Failed to close cache: %v", err)
	}

	c2, err := New(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create new cache: %v", err)
	}
	cachedData, ok, err = c2.Get(url)
	if err != nil {
		t.Fatalf("Failed to get data from new cache: %v", err)
	}
	if !ok {
		t.Fatal("Expected data to be in new cache, but it wasn't")
	}
	if string(cachedData) != string(data) {
		t.Errorf("Expected data to be '%s', but got '%s'", string(data), string(cachedData))
	}
}

func TestCache_Clear(t *testing.T) {
	cacheDir := t.TempDir()
	c, err := New(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Put some data in the cache
	url := "https://example.com"
	data := []byte("hello world")
	err = c.Put(url, data)
	if err != nil {
		t.Fatalf("Failed to put data in cache: %v", err)
	}
	err = c.Close()
	if err != nil {
		t.Fatalf("Failed to close cache: %v", err)
	}

	// Clear the cache
	err = c.Clear()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Check that the cache directory is gone
	_, err = os.Stat(cacheDir)
	if !os.IsNotExist(err) {
		t.Fatal("Expected cache directory to be gone, but it wasn't")
	}
}

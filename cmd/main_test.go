package cmd

import (
	"bytes"
	"io"
	"net/url"
	"os"
	"testing"

	"github.com/Snider/Borg/pkg/storage"
)

// MockStorage is a mock implementation of the Storage interface for testing.
type MockStorage struct {
	WriteFunc func(path string, data io.Reader) error
	ReadFunc  func(path string) (io.ReadCloser, error)
	ListFunc  func(path string) ([]string, error)
}

func (m *MockStorage) Write(path string, data io.Reader) error {
	if m.WriteFunc != nil {
		return m.WriteFunc(path, data)
	}
	return nil
}

func (m *MockStorage) Read(path string) (io.ReadCloser, error) {
	if m.ReadFunc != nil {
		return m.ReadFunc(path)
	}
	return io.NopCloser(bytes.NewReader([]byte{})), nil
}

func (m *MockStorage) List(path string) ([]string, error) {
	if m.ListFunc != nil {
		return m.ListFunc(path)
	}
	return []string{}, nil
}

func TestMain(m *testing.M) {
	// Mock the storage backend for all tests in this package
	originalNewStorage := storage.NewStorage
	storage.NewStorage = func(u *url.URL) (storage.Storage, error) {
		if u.Scheme == "mock" {
			return &MockStorage{}, nil
		}
		return originalNewStorage(u)
	}
	os.Exit(m.Run())
}

package storage

import (
	"fmt"
	"io"
	"net/url"
)

// Storage is an interface for a storage backend.
type Storage interface {
	// Write writes data to the given path.
	Write(path string, data io.Reader) error
	// Read reads data from the given path.
	Read(path string) (io.ReadCloser, error)
	// List lists the contents of the given path.
	List(path string) ([]string, error)
}

// NewStorage creates a new storage backend based on the URL scheme.
var NewStorage = newStorage

func newStorage(u *url.URL) (Storage, error) {
	switch u.Scheme {
	case "s3":
		return NewS3Storage(u.Host)
	case "r2":
		return nil, fmt.Errorf("r2 storage not implemented")
	case "b2":
		return nil, fmt.Errorf("b2 storage not implemented")
	case "file":
		return nil, fmt.Errorf("file storage not implemented")
	case "mock":
		return nil, nil // Handled in tests
	default:
		return nil, fmt.Errorf("unsupported storage scheme: %s", u.Scheme)
	}
}
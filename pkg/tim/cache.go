package tim

import (
	"os"
	"path/filepath"
	"strings"
)

// Cache provides encrypted storage for TIM containers.
// It stores TIMs as .stim files in a directory, encrypted with
// ChaCha20-Poly1305 using a shared password.
type Cache struct {
	Dir      string
	Password string
}

// NewCache creates a cache in the given directory.
// The directory will be created if it doesn't exist.
func NewCache(dir, password string) (*Cache, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Cache{Dir: dir, Password: password}, nil
}

// Store encrypts and saves a TIM to the cache.
func (c *Cache) Store(name string, m *TerminalIsolationMatrix) error {
	data, err := m.ToSigil(c.Password)
	if err != nil {
		return err
	}
	path := filepath.Join(c.Dir, name+".stim")
	return os.WriteFile(path, data, 0600)
}

// Load retrieves and decrypts a TIM from the cache.
func (c *Cache) Load(name string) (*TerminalIsolationMatrix, error) {
	path := filepath.Join(c.Dir, name+".stim")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return FromSigil(data, c.Password)
}

// Delete removes a TIM from the cache.
func (c *Cache) Delete(name string) error {
	path := filepath.Join(c.Dir, name+".stim")
	return os.Remove(path)
}

// Exists checks if a TIM exists in the cache.
func (c *Cache) Exists(name string) bool {
	path := filepath.Join(c.Dir, name+".stim")
	_, err := os.Stat(path)
	return err == nil
}

// List returns all cached TIM names.
func (c *Cache) List() ([]string, error) {
	entries, err := os.ReadDir(c.Dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".stim") {
			names = append(names, strings.TrimSuffix(e.Name(), ".stim"))
		}
	}
	return names, nil
}

// Run loads and executes a TIM from the cache using runc.
func (c *Cache) Run(name string) error {
	path := filepath.Join(c.Dir, name+".stim")
	return RunEncrypted(path, c.Password)
}

// Size returns the size of a cached TIM in bytes.
func (c *Cache) Size(name string) (int64, error) {
	path := filepath.Join(c.Dir, name+".stim")
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

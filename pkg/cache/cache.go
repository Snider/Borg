package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	indexFileName   = "index.json"
	storageDirName  = "sha256"
)

// Cache provides a content-addressable storage for web content.
type Cache struct {
	dir     string
	index   map[string]string
	mutex   sync.RWMutex
}

// New creates a new Cache instance.
func New(dir string) (*Cache, error) {
	storageDir := filepath.Join(dir, storageDirName)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache storage directory: %w", err)
	}

	cache := &Cache{
		dir:     dir,
		index:   make(map[string]string),
	}

	if err := cache.loadIndex(); err != nil {
		return nil, fmt.Errorf("failed to load cache index: %w", err)
	}

	return cache, nil
}

// Get retrieves content from the cache for a given URL.
func (c *Cache) Get(url string) ([]byte, bool, error) {
	c.mutex.RLock()
	hash, ok := c.index[url]
	c.mutex.RUnlock()

	if !ok {
		return nil, false, nil
	}

	path := c.getStoragePath(hash)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to read from cache: %w", err)
	}

	return data, true, nil
}

// Put adds content to the cache for a given URL.
func (c *Cache) Put(url string, data []byte) error {
	hashBytes := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", hashBytes)

	path := c.getStoragePath(hash)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	c.mutex.Lock()
	c.index[url] = hash
	c.mutex.Unlock()

	return nil
}

// Close saves the index file.
func (c *Cache) Close() error {
	return c.saveIndex()
}

// Clear removes the cache directory.
func (c *Cache) Clear() error {
	return os.RemoveAll(c.dir)
}

// Dir returns the cache directory.
func (c *Cache) Dir() string {
	return c.dir
}

// Size returns the total size of the cache.
func (c *Cache) Size() (int64, error) {
	var size int64
	err := filepath.Walk(c.dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// NumEntries returns the number of entries in the cache.
func (c *Cache) NumEntries() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.index)
}


func (c *Cache) getStoragePath(hash string) string {
	return filepath.Join(c.dir, storageDirName, hash[:2], hash)
}

func (c *Cache) loadIndex() error {
	indexPath := filepath.Join(c.dir, indexFileName)
	file, err := os.Open(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return decoder.Decode(&c.index)
}

func (c *Cache) saveIndex() error {
	indexPath := filepath.Join(c.dir, indexFileName)
	file, err := os.Create(indexPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return encoder.Encode(c.index)
}

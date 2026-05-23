package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type MetadataCache struct {
	mu       sync.RWMutex
	filePath string
	Entries  map[string]*CacheEntry `json:"entries"`
}

type CacheEntry struct {
	Weight     int64     `json:"weight"`
	Resolved   string    `json:"resolved"`
	Integrity  string    `json:"integrity"`
	CachedAt   time.Time `json:"cached_at"`
}

func NewMetadataCache(cacheDir string) (*MetadataCache, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(cacheDir, "metadata.json")
	c := &MetadataCache{
		filePath: path,
		Entries:  make(map[string]*CacheEntry),
	}

	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, c)
	}

	return c, nil
}

func (c *MetadataCache) Get(name string) *CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Entries[name]
}

func (c *MetadataCache) Set(name string, entry *CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Entries[name] = entry
}

func (c *MetadataCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.filePath, data, 0644)
}

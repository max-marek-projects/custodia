// Package client provides a local on-disk cache of encrypted secrets.
//
// The cache lets the CLI serve previously fetched secrets when the server
// is unreachable. Only ciphertext returned by the server is stored: the
// payload is encrypted with a key derived from the user's master password,
// so the cache file is useless without the correct password.
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// cacheFileName is the name of the file used to persist the secret cache.
// It lives next to the tokens file in the user's config directory.
const cacheFileName = "cache.json"

// cachedSecret is a single cache entry: the encrypted payload returned by
// the server, plus enough metadata to identify the entry and decrypt it
// with the user's password.
type cachedSecret struct {
	UserID   int64             `json:"user_id"`
	DataType int32             `json:"data_type"`
	Name     string            `json:"name"`
	Version  uint64            `json:"version"`
	Salt     []byte            `json:"salt"`
	IV       []byte            `json:"iv"`
	Data     []byte            `json:"data"`
	Metadata map[string]string `json:"metadata,omitempty"`
	CachedAt time.Time         `json:"cached_at"`
}

// cacheFile is the on-disk representation of the cache.
type cacheFile struct {
	Entries []cachedSecret `json:"entries"`
}

// secretCache is a thread-safe, file-backed cache of encrypted secrets.
type secretCache struct {
	mu      sync.RWMutex
	folder  string
	logger  *slog.Logger
	entries map[string]cachedSecret
}

// newSecretCache creates an empty cache. Use Load to read entries from disk.
func newSecretCache(folder string, logger *slog.Logger) (*secretCache, error) {
	secretCache := &secretCache{
		folder:  folder,
		logger:  logger,
		entries: make(map[string]cachedSecret),
	}
	err := secretCache.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load cache file from disk: %w", err)
	}
	return secretCache, nil
}

// cacheKey builds the map key for a cache entry.
func cacheKey(userID int64, name string, version uint64) string {
	return fmt.Sprintf("%d:%s:%d", userID, name, version)
}

// cachePath returns the full path to the cache file, creating the parent
// directory with restrictive permissions if necessary.
func (c *secretCache) cachePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user configuration directory: %w", err)
	}
	folder := c.folder
	if folder == "" {
		folder = "custodia"
	}
	dir := filepath.Join(configDir, folder)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}
	return filepath.Join(dir, cacheFileName), nil
}

// Load reads the cache file from disk. A missing file is not an error.
// A corrupt file is logged and treated as empty.
func (c *secretCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	path, err := c.cachePath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.logger.Debug("secret cache file not found", slog.String("path", path))
			return nil
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		c.logger.Warn("failed to parse cache file, starting with empty cache",
			slog.String("path", path), slog.Any("error", err))
		return nil
	}

	for _, e := range cf.Entries {
		c.entries[cacheKey(e.UserID, e.Name, e.Version)] = e
	}
	c.logger.Debug("secret cache loaded",
		slog.String("path", path), slog.Int("entries", len(c.entries)))
	return nil
}

// saveLocked persists the current in-memory cache to disk.
// The caller must hold c.mu.
func (c *secretCache) saveLocked() error {
	path, err := c.cachePath()
	if err != nil {
		return err
	}
	cf := cacheFile{Entries: make([]cachedSecret, 0, len(c.entries))}
	for _, e := range c.entries {
		cf.Entries = append(cf.Entries, e)
	}
	data, err := json.MarshalIndent(cf, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}
	return nil
}

// Put stores an entry in the cache and persists the cache to disk.
func (c *secretCache) Put(entry cachedSecret) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry.CachedAt = time.Now()
	c.entries[cacheKey(entry.UserID, entry.Name, entry.Version)] = entry
	return c.saveLocked()
}

// Get returns a cached entry for the given (userID, dataType, name, version).
// Version is matched exactly: a request for the latest version (version == 0)
// hits the entry cached by an earlier "latest" lookup, and a request for an
// explicit version hits the entry cached by an earlier explicit lookup.
func (c *secretCache) Get(
	userID int64,
	name string,
	version uint64,
) (cachedSecret, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[cacheKey(userID, name, version)]
	return e, ok
}

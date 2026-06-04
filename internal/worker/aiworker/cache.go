package aiworker

import (
	"crypto/sha256" // produces the content-addressed cache key
	"encoding/hex"  // renders the hash bytes as a hex string
	"encoding/json" // serialises and deserialises cache entries
	"os"            // creates directories and reads/writes files
	"path/filepath" // joins the cache directory and key into a file path
)

// cacheEntry is the on-disk format for a single embedding vector.
type cacheEntry struct {
	Text   string    `json:"text"`   // original text for human inspection
	Vector []float64 `json:"vector"` // embedding vector produced by the AI worker
}

// EmbeddingCache stores embedding vectors on disk so repeated calls for the same
// text skip the network round-trip to the AI worker. Each entry is a small JSON
// file named by the SHA-256 of the original text. The zero value is not usable;
// construct with NewEmbeddingCache.
type EmbeddingCache struct {
	dir string // directory under which cache files are stored
}

// NewEmbeddingCache returns an EmbeddingCache that persists files under dir.
// dir is created on first write; pass an empty string to get a no-op cache.
func NewEmbeddingCache(dir string) *EmbeddingCache {
	return &EmbeddingCache{dir: dir}
}

// Get returns the cached vector for text and true, or nil, false on a miss or any
// read error. Errors are silently discarded: a cache miss is always safe.
func (c *EmbeddingCache) Get(text string) ([]float64, bool) {
	if c.dir == "" {
		return nil, false
	}
	data, err := os.ReadFile(c.entryPath(text))
	if err != nil {
		return nil, false
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}
	return entry.Vector, true
}

// Set writes vector for text to disk. It creates the cache directory if needed.
// An empty dir is a no-op. Write errors are returned so callers can log them,
// but a Set failure never affects the vector already produced by the worker.
func (c *EmbeddingCache) Set(text string, vector []float64) error {
	if c.dir == "" {
		return nil
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	entry := cacheEntry{Text: text, Vector: vector}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(c.entryPath(text), data, 0o644)
}

// entryPath returns the file path for the given text's cache entry.
func (c *EmbeddingCache) entryPath(text string) string {
	sum := sha256.Sum256([]byte(text))
	key := hex.EncodeToString(sum[:])
	return filepath.Join(c.dir, key+".json")
}

// CacheKey returns the hex SHA-256 key for text. It is exported so callers and
// tests can compute expected paths without constructing a full EmbeddingCache.
func CacheKey(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

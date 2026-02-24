package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Cache provides content-addressed file storage.
// Files are stored by their SHA256 hash and verified on retrieval.
type Cache struct {
	dir string
}

// New creates a Cache at the given directory.
// The directory is created if it does not exist.
func New(dir string) (*Cache, error) {
	objDir := filepath.Join(dir, "objects")
	if err := os.MkdirAll(objDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory %s: %w", objDir, err)
	}
	return &Cache{dir: dir}, nil
}

// DefaultDir returns the default cache directory.
// Uses XDG_CACHE_HOME if set, otherwise ~/.cache/agent-sync.
func DefaultDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "agent-sync")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		if runtime.GOOS == "windows" {
			return filepath.Join(os.TempDir(), "agent-sync-cache")
		}
		return filepath.Join("/tmp", "agent-sync-cache")
	}
	return filepath.Join(home, ".cache", "agent-sync")
}

// Get retrieves a cached file by its SHA256 hash.
// Returns the content and true if found and verified.
// Returns nil, false if not cached.
// Returns error if cached but hash verification fails (corruption).
func (c *Cache) Get(hash string) ([]byte, bool, error) {
	path := c.objectPath(hash)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading cache entry %s: %w", hash, err)
	}

	// Verify hash (Section 8.4: verified before use).
	actual := computeHash(data)
	if actual != hash {
		// Self-healing: remove corrupt entry.
		_ = os.Remove(path)
		return nil, false, nil
	}

	return data, true, nil
}

// Put stores content in the cache by its SHA256 hash.
// Verifies the content matches the hash before storing.
// No-op if already cached.
func (c *Cache) Put(hash string, content []byte) error {
	// Verify content matches the declared hash.
	actual := computeHash(content)
	if actual != hash {
		return fmt.Errorf("cache put: content hash %s does not match declared hash %s", actual, hash)
	}

	path := c.objectPath(hash)

	// Already cached — immutable, no overwrite needed.
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	// Create subdirectory.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating cache subdirectory: %w", err)
	}

	// Atomic write: temp file + rename.
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("creating cache temp file: %w", err)
	}
	tmpPath := tmp.Name()

	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		return fmt.Errorf("writing cache temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("syncing cache temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing cache temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming cache temp file: %w", err)
	}

	success = true
	return nil
}

// Has checks if a hash exists in the cache without reading content.
func (c *Cache) Has(hash string) bool {
	_, err := os.Stat(c.objectPath(hash))
	return err == nil
}

// Size returns the total size of the cache in bytes.
func (c *Cache) Size() (int64, error) {
	var total int64
	err := filepath.Walk(c.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// Path returns the cache directory path.
func (c *Cache) Path() string {
	return c.dir
}

func (c *Cache) objectPath(hash string) string {
	if len(hash) < 2 {
		return filepath.Join(c.dir, "objects", hash)
	}
	return filepath.Join(c.dir, "objects", hash[:2], hash)
}

// ComputeHash computes the SHA256 hash of content and returns the hex string.
func ComputeHash(content []byte) string {
	return computeHash(content)
}

func computeHash(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

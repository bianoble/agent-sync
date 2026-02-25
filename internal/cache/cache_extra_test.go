package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestObjectPathShortHash(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Hash with fewer than 2 characters should not use subdirectory.
	path := c.objectPath("a")
	expected := filepath.Join(dir, "objects", "a")
	if path != expected {
		t.Errorf("objectPath(%q) = %q, want %q", "a", path, expected)
	}

	// Empty hash.
	path = c.objectPath("")
	expected = filepath.Join(dir, "objects", "")
	if path != expected {
		t.Errorf("objectPath(%q) = %q, want %q", "", path, expected)
	}
}

func TestGetReadError(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a directory where the cache file should be, causing a read error.
	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	objPath := c.objectPath(hash)
	if mkdirErr := os.MkdirAll(objPath, 0755); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}

	_, _, err = c.Get(hash)
	if err == nil {
		t.Fatal("expected error when reading a directory as a file")
	}
}

func TestNewCreatesDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test unreliable as root")
	}

	// Try to create cache in a read-only directory.
	dir := t.TempDir()
	readOnly := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(readOnly, 0555); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chmod(readOnly, 0755)
	}()

	_, err := New(filepath.Join(readOnly, "nested", "cache"))
	if err == nil {
		t.Fatal("expected error creating cache in read-only dir")
	}
}

func TestSizeEmptyCache(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	size, err := c.Size()
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	if size != 0 {
		t.Errorf("expected 0 size for empty cache, got %d", size)
	}
}

func TestComputeHashPublic(t *testing.T) {
	h1 := ComputeHash([]byte("test"))
	h2 := ComputeHash([]byte("test"))
	if h1 != h2 {
		t.Error("same content should produce same hash")
	}

	h3 := ComputeHash([]byte("different"))
	if h1 == h3 {
		t.Error("different content should produce different hashes")
	}
}

func TestDefaultDirWithoutXDG(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	got := DefaultDir()
	// Should fall back to ~/.cache/agent-sync or temp dir.
	if got == "" {
		t.Error("DefaultDir should not be empty")
	}
	if !filepath.IsAbs(got) {
		t.Errorf("DefaultDir should return absolute path, got %q", got)
	}
}

func TestSizeWalkError(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Remove cache dir to cause walk error.
	_ = os.RemoveAll(dir)

	_, err = c.Size()
	if err == nil {
		t.Fatal("expected error when cache dir is removed")
	}
}

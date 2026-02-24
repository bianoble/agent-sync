package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPutAndGet(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	content := []byte("hello world")
	hash := ComputeHash(content)

	if putErr := c.Put(hash, content); putErr != nil {
		t.Fatalf("Put: %v", putErr)
	}

	got, found, err := c.Get(hash)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected cache hit")
	}
	if string(got) != "hello world" {
		t.Errorf("got %q", string(got))
	}
}

func TestGetMiss(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, found, err := c.Get("nonexistent_hash")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatal("expected cache miss")
	}
}

func TestPutWrongHash(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Put("wrong_hash", []byte("content"))
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}
}

func TestPutIdempotent(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("idempotent")
	hash := ComputeHash(content)

	// Put twice — should not error.
	if putErr := c.Put(hash, content); putErr != nil {
		t.Fatalf("first Put: %v", putErr)
	}
	if putErr := c.Put(hash, content); putErr != nil {
		t.Fatalf("second Put: %v", putErr)
	}
}

func TestCorruptCacheEntry(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("original content")
	hash := ComputeHash(content)

	if putErr := c.Put(hash, content); putErr != nil {
		t.Fatal(putErr)
	}

	// Corrupt the cache entry.
	objPath := c.objectPath(hash)
	if writeErr := os.WriteFile(objPath, []byte("corrupted"), 0644); writeErr != nil {
		t.Fatal(writeErr)
	}

	// Get should detect corruption and return miss (self-healing).
	_, found, err := c.Get(hash)
	if err != nil {
		t.Fatalf("Get should not error on corruption: %v", err)
	}
	if found {
		t.Fatal("expected cache miss after corruption")
	}

	// Corrupt file should be cleaned up.
	if _, statErr := os.Stat(objPath); !os.IsNotExist(statErr) {
		t.Error("corrupt cache entry should be removed")
	}
}

func TestHas(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("exists")
	hash := ComputeHash(content)

	if c.Has(hash) {
		t.Fatal("expected Has=false before Put")
	}

	if putErr := c.Put(hash, content); putErr != nil {
		t.Fatal(putErr)
	}

	if !c.Has(hash) {
		t.Fatal("expected Has=true after Put")
	}
}

func TestSize(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("some content for size test")
	hash := ComputeHash(content)
	if putErr := c.Put(hash, content); putErr != nil {
		t.Fatal(putErr)
	}

	size, err := c.Size()
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	if size <= 0 {
		t.Errorf("expected positive size, got %d", size)
	}
}

func TestPath(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	if c.Path() != dir {
		t.Errorf("Path = %q, want %q", c.Path(), dir)
	}
}

func TestDefaultDir(t *testing.T) {
	// Test with XDG_CACHE_HOME set.
	original := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		if err := os.Setenv("XDG_CACHE_HOME", original); err != nil {
			t.Errorf("failed to restore XDG_CACHE_HOME: %v", err)
		}
	}()

	if err := os.Setenv("XDG_CACHE_HOME", "/custom/cache"); err != nil {
		t.Fatalf("failed to set XDG_CACHE_HOME: %v", err)
	}
	got := DefaultDir()
	want := filepath.Join("/custom/cache", "agent-sync")
	if got != want {
		t.Errorf("with XDG_CACHE_HOME: got %q, want %q", got, want)
	}

	// Test without XDG_CACHE_HOME.
	if err := os.Unsetenv("XDG_CACHE_HOME"); err != nil {
		t.Fatalf("failed to unset XDG_CACHE_HOME: %v", err)
	}
	got = DefaultDir()
	if got == "" {
		t.Error("DefaultDir should not be empty")
	}
}

func TestObjectPathLayout(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	hash := "abcdef1234567890"
	path := c.objectPath(hash)
	// Should use first 2 chars as subdirectory.
	expected := filepath.Join(dir, "objects", "ab", hash)
	if path != expected {
		t.Errorf("objectPath = %q, want %q", path, expected)
	}
}

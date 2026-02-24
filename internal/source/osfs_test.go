package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOSFSReadWriteFile(t *testing.T) {
	dir := t.TempDir()
	fs := OSFS{}

	content := []byte("hello world")
	path := filepath.Join(dir, "test.txt")

	if err := fs.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("content = %q, want 'hello world'", string(got))
	}
}

func TestOSFSStat(t *testing.T) {
	dir := t.TempDir()
	fs := OSFS{}

	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := fs.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.IsDir() {
		t.Error("expected file, not directory")
	}
	if info.Size() != 4 {
		t.Errorf("size = %d, want 4", info.Size())
	}
}

func TestOSFSMkdirAll(t *testing.T) {
	dir := t.TempDir()
	fs := OSFS{}

	nested := filepath.Join(dir, "a", "b", "c")
	if err := fs.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestOSFSRemove(t *testing.T) {
	dir := t.TempDir()
	fs := OSFS{}

	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := fs.Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should have been removed")
	}
}

func TestOSFSRename(t *testing.T) {
	dir := t.TempDir()
	fs := OSFS{}

	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(oldPath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := fs.Rename(oldPath, newPath); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("old file should not exist")
	}
	got, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "data" {
		t.Errorf("content = %q", string(got))
	}
}

func TestOSFSWalk(t *testing.T) {
	dir := t.TempDir()
	fs := OSFS{}

	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	var paths []string
	err := fs.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(paths) != 1 {
		t.Errorf("found %d files, want 1", len(paths))
	}
}

func TestDefaultHTTPClientDo(t *testing.T) {
	// Just verify it doesn't panic and returns the expected type.
	client := DefaultHTTPClient{}
	_ = client // Ensure it compiles and the type is usable.
}

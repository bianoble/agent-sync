package sandbox

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidatePathWithinRoot(t *testing.T) {
	root := t.TempDir()

	resolved, err := ValidatePath(root, "subdir/file.txt")
	if err != nil {
		t.Fatalf("ValidatePath: %v", err)
	}

	realRoot, _ := filepath.EvalSymlinks(root)
	expected := filepath.Join(realRoot, "subdir/file.txt")
	if resolved != expected {
		t.Errorf("got %q, want %q", resolved, expected)
	}
}

func TestValidatePathRejectsDotDot(t *testing.T) {
	root := t.TempDir()

	_, err := ValidatePath(root, "../escape.txt")
	if err == nil {
		t.Fatal("expected error for .. escape")
	}
	if !strings.Contains(err.Error(), "outside the project root") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidatePathRejectsDotDotNested(t *testing.T) {
	root := t.TempDir()

	_, err := ValidatePath(root, "subdir/../../escape.txt")
	if err == nil {
		t.Fatal("expected error for nested .. escape")
	}
	if !strings.Contains(err.Error(), "outside the project root") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidatePathRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not reliable on Windows")
	}

	root := t.TempDir()
	outsideDir := t.TempDir()

	// Create a symlink inside root pointing outside.
	symlink := filepath.Join(root, "escape-link")
	if err := os.Symlink(outsideDir, symlink); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}

	_, err := ValidatePath(root, "escape-link/file.txt")
	if err == nil {
		t.Fatal("expected error for symlink escape")
	}
	if !strings.Contains(err.Error(), "outside the project root") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidatePathAllowsInternalSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not reliable on Windows")
	}

	root := t.TempDir()
	// Create a real target directory.
	realDir := filepath.Join(root, "real")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a symlink inside root pointing to another location inside root.
	symlink := filepath.Join(root, "link")
	if err := os.Symlink(realDir, symlink); err != nil {
		t.Fatal(err)
	}

	resolved, err := ValidatePath(root, "link/file.txt")
	if err != nil {
		t.Fatalf("ValidatePath should allow internal symlinks: %v", err)
	}

	realRoot, _ := filepath.EvalSymlinks(root)
	expected := filepath.Join(realRoot, "real", "file.txt")
	if resolved != expected {
		t.Errorf("got %q, want %q", resolved, expected)
	}
}

func TestSafeWriteCreatesFile(t *testing.T) {
	root := t.TempDir()
	content := []byte("hello world")

	if err := SafeWrite(root, "subdir/test.txt", content, 0644); err != nil {
		t.Fatalf("SafeWrite: %v", err)
	}

	realRoot, _ := filepath.EvalSymlinks(root)
	written, err := os.ReadFile(filepath.Join(realRoot, "subdir/test.txt"))
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(written) != "hello world" {
		t.Errorf("content = %q, want %q", string(written), "hello world")
	}
}

func TestSafeWriteOverwritesExisting(t *testing.T) {
	root := t.TempDir()

	if err := SafeWrite(root, "file.txt", []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SafeWrite(root, "file.txt", []byte("updated"), 0644); err != nil {
		t.Fatal(err)
	}

	realRoot, _ := filepath.EvalSymlinks(root)
	data, _ := os.ReadFile(filepath.Join(realRoot, "file.txt"))
	if string(data) != "updated" {
		t.Errorf("content = %q, want %q", string(data), "updated")
	}
}

func TestSafeWriteRejectsEscape(t *testing.T) {
	root := t.TempDir()
	err := SafeWrite(root, "../escape.txt", []byte("bad"), 0644)
	if err == nil {
		t.Fatal("expected error for escape attempt")
	}
}

func TestSafeRemove(t *testing.T) {
	root := t.TempDir()

	// Create a file to remove.
	if err := SafeWrite(root, "to-delete.txt", []byte("bye"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SafeRemove(root, "to-delete.txt"); err != nil {
		t.Fatalf("SafeRemove: %v", err)
	}

	realRoot, _ := filepath.EvalSymlinks(root)
	if _, err := os.Stat(filepath.Join(realRoot, "to-delete.txt")); !os.IsNotExist(err) {
		t.Error("file should be removed")
	}
}

func TestSafeRemoveRejectsEscape(t *testing.T) {
	root := t.TempDir()
	err := SafeRemove(root, "../escape.txt")
	if err == nil {
		t.Fatal("expected error for escape attempt")
	}
}

func TestSafeMkdirAll(t *testing.T) {
	root := t.TempDir()

	if err := SafeMkdirAll(root, "a/b/c", 0755); err != nil {
		t.Fatalf("SafeMkdirAll: %v", err)
	}

	realRoot, _ := filepath.EvalSymlinks(root)
	info, err := os.Stat(filepath.Join(realRoot, "a/b/c"))
	if err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("should be a directory")
	}
}

package sandbox

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidatePathRootItself(t *testing.T) {
	root := t.TempDir()
	resolved, err := ValidatePath(root, ".")
	if err != nil {
		t.Fatalf("ValidatePath for root itself: %v", err)
	}

	realRoot, _ := filepath.EvalSymlinks(root)
	if resolved != realRoot {
		t.Errorf("got %q, want %q", resolved, realRoot)
	}
}

func TestValidatePathAbsoluteEscape(t *testing.T) {
	root := t.TempDir()
	// Path with many ../ to try escaping
	_, err := ValidatePath(root, "a/b/c/../../../../escape.txt")
	if err == nil {
		t.Fatal("expected error for deep .. escape")
	}
	if !strings.Contains(err.Error(), "outside the project root") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidatePathInvalidRoot(t *testing.T) {
	// Use a non-existent root directory.
	_, err := ValidatePath("/nonexistent-root-dir-12345", "file.txt")
	if err == nil {
		t.Fatal("expected error for non-existent root")
	}
}

func TestSafeWritePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission test not reliable on Windows")
	}

	root := t.TempDir()
	content := []byte("test content")

	if err := SafeWrite(root, "test.txt", content, 0600); err != nil {
		t.Fatalf("SafeWrite: %v", err)
	}

	realRoot, _ := filepath.EvalSymlinks(root)
	info, err := os.Stat(filepath.Join(realRoot, "test.txt"))
	if err != nil {
		t.Fatal(err)
	}

	// Check permissions were set.
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected permission 0600, got %04o", perm)
	}
}

func TestSafeRemoveNonexistent(t *testing.T) {
	root := t.TempDir()
	err := SafeRemove(root, "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error removing nonexistent file")
	}
}

func TestSafeMkdirAllRejectsEscape(t *testing.T) {
	root := t.TempDir()
	err := SafeMkdirAll(root, "../escape", 0755)
	if err == nil {
		t.Fatal("expected error for escape attempt")
	}
}

func TestSafeMkdirAllExisting(t *testing.T) {
	root := t.TempDir()

	// Create the directory first.
	if err := SafeMkdirAll(root, "already/exists", 0755); err != nil {
		t.Fatalf("first SafeMkdirAll: %v", err)
	}

	// Should be idempotent.
	if err := SafeMkdirAll(root, "already/exists", 0755); err != nil {
		t.Fatalf("second SafeMkdirAll: %v", err)
	}
}

func TestSafeWriteParentEscape(t *testing.T) {
	root := t.TempDir()
	err := SafeWrite(root, "../escape/file.txt", []byte("bad"), 0644)
	if err == nil {
		t.Fatal("expected error for parent escape")
	}
}

func TestSafeWriteDeepNested(t *testing.T) {
	root := t.TempDir()
	content := []byte("deep content")

	if err := SafeWrite(root, "a/b/c/d/e/file.txt", content, 0644); err != nil {
		t.Fatalf("SafeWrite deep nested: %v", err)
	}

	realRoot, _ := filepath.EvalSymlinks(root)
	data, err := os.ReadFile(filepath.Join(realRoot, "a/b/c/d/e/file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "deep content" {
		t.Errorf("content = %q, want %q", string(data), "deep content")
	}
}

func TestResolveExistingPathFullyExists(t *testing.T) {
	dir := t.TempDir()
	// Create a file.
	filePath := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	resolved, err := resolveExistingPath(filePath)
	if err != nil {
		t.Fatalf("resolveExistingPath: %v", err)
	}

	// Should resolve to the real path.
	realDir, _ := filepath.EvalSymlinks(dir)
	expected := filepath.Join(realDir, "existing.txt")
	if resolved != expected {
		t.Errorf("got %q, want %q", resolved, expected)
	}
}

func TestResolveExistingPathPartiallyExists(t *testing.T) {
	dir := t.TempDir()
	// Path where dir exists but file does not.
	partial := filepath.Join(dir, "nonexistent-file.txt")

	resolved, err := resolveExistingPath(partial)
	if err != nil {
		t.Fatalf("resolveExistingPath: %v", err)
	}

	realDir, _ := filepath.EvalSymlinks(dir)
	expected := filepath.Join(realDir, "nonexistent-file.txt")
	if resolved != expected {
		t.Errorf("got %q, want %q", resolved, expected)
	}
}

func TestResolveExistingPathDeeplyNonexistent(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c", "file.txt")

	resolved, err := resolveExistingPath(deep)
	if err != nil {
		t.Fatalf("resolveExistingPath: %v", err)
	}

	realDir, _ := filepath.EvalSymlinks(dir)
	expected := filepath.Join(realDir, "a", "b", "c", "file.txt")
	if resolved != expected {
		t.Errorf("got %q, want %q", resolved, expected)
	}
}

func TestValidatePathWithSymlinkInMiddle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not reliable on Windows")
	}

	root := t.TempDir()
	// Create real/subdir structure.
	realDir := filepath.Join(root, "real", "subdir")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create symlink: root/link -> root/real
	symlink := filepath.Join(root, "link")
	if err := os.Symlink(filepath.Join(root, "real"), symlink); err != nil {
		t.Fatal(err)
	}

	// Accessing through the symlink should work since it stays in root.
	resolved, err := ValidatePath(root, "link/subdir/file.txt")
	if err != nil {
		t.Fatalf("ValidatePath through internal symlink: %v", err)
	}

	realRoot, _ := filepath.EvalSymlinks(root)
	expected := filepath.Join(realRoot, "real", "subdir", "file.txt")
	if resolved != expected {
		t.Errorf("got %q, want %q", resolved, expected)
	}
}

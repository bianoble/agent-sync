package source

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bianoble/agent-sync/internal/config"
)

func TestLocalResolverDirectory(t *testing.T) {
	root := t.TempDir()
	agentsDir := filepath.Join(root, "agents", "standards")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "naming.md"), []byte("# Naming\n"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "testing.md"), []byte("# Testing\n"), 0644)

	r := &LocalResolver{}
	src := config.Source{Name: "standards", Type: "local", Path: "./agents/standards/"}

	resolved, err := r.Resolve(context.Background(), src, root)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if resolved.Name != "standards" {
		t.Errorf("name = %q", resolved.Name)
	}
	if resolved.Type != "local" {
		t.Errorf("type = %q", resolved.Type)
	}
	if len(resolved.Files) != 2 {
		t.Errorf("files count = %d, want 2: %v", len(resolved.Files), resolved.Files)
	}
	if _, ok := resolved.Files["naming.md"]; !ok {
		t.Errorf("expected naming.md in files")
	}
	if _, ok := resolved.Files["testing.md"]; !ok {
		t.Errorf("expected testing.md in files")
	}
}

func TestLocalResolverSingleFile(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "policy.md"), []byte("# Policy\n"), 0644)

	r := &LocalResolver{}
	src := config.Source{Name: "policy", Type: "local", Path: "./policy.md"}

	resolved, err := r.Resolve(context.Background(), src, root)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(resolved.Files) != 1 {
		t.Fatalf("files count = %d, want 1", len(resolved.Files))
	}
	if _, ok := resolved.Files["policy.md"]; !ok {
		t.Errorf("expected policy.md in files")
	}
}

func TestLocalResolverMissingPath(t *testing.T) {
	r := &LocalResolver{}
	_, err := r.Resolve(context.Background(), config.Source{Name: "test", Type: "local"}, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLocalResolverNonexistentPath(t *testing.T) {
	r := &LocalResolver{}
	src := config.Source{Name: "test", Type: "local", Path: "./does-not-exist/"}
	_, err := r.Resolve(context.Background(), src, t.TempDir())
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestLocalResolverEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "empty"), 0755)

	r := &LocalResolver{}
	src := config.Source{Name: "empty", Type: "local", Path: "./empty/"}

	_, err := r.Resolve(context.Background(), src, root)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no files found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLocalResolverRejectsEscapePath(t *testing.T) {
	root := t.TempDir()
	r := &LocalResolver{}
	src := config.Source{Name: "escape", Type: "local", Path: "../../etc/passwd"}

	_, err := r.Resolve(context.Background(), src, root)
	if err == nil {
		t.Fatal("expected error for path escape")
	}
}

func TestLocalResolverSkipsHiddenFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "visible.md"), []byte("visible"), 0644)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("hidden"), 0644)

	r := &LocalResolver{}
	src := config.Source{Name: "test", Type: "local", Path: "./src/"}

	resolved, err := r.Resolve(context.Background(), src, root)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolved.Files) != 1 {
		t.Errorf("expected 1 file (hidden should be skipped), got %d: %v", len(resolved.Files), resolved.Files)
	}
}

func TestLocalFetchWithRoot(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "agents")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "rules.md"), []byte("# Rules\n"), 0644)

	r := &LocalResolver{}
	src := config.Source{Name: "test", Type: "local", Path: "./agents/"}

	resolved, err := r.Resolve(context.Background(), src, root)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	fetched, err := r.FetchWithRoot(context.Background(), resolved, root)
	if err != nil {
		t.Fatalf("FetchWithRoot: %v", err)
	}
	if len(fetched) != 1 {
		t.Fatalf("expected 1 file, got %d", len(fetched))
	}
	if string(fetched[0].Content) != "# Rules\n" {
		t.Errorf("content = %q", string(fetched[0].Content))
	}
}

func TestLocalFetchHashMismatch(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "file.md"), []byte("original"), 0644)

	r := &LocalResolver{}
	resolved := &ResolvedSource{
		Name:  "test",
		Type:  "local",
		Path:  "./src/",
		Files: map[string]string{"file.md": "wrong_hash"},
	}

	_, err := r.FetchWithRoot(context.Background(), resolved, root)
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}
	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("unexpected error: %v", err)
	}
}

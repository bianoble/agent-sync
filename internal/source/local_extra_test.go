package source

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bianoble/agent-sync/internal/config"
)

func TestLocalFetchMissingPath(t *testing.T) {
	r := &LocalResolver{}
	resolved := &ResolvedSource{
		Name: "test",
		Type: "local",
		// Path intentionally empty.
		Files: map[string]string{"file.md": "hash"},
	}

	_, err := r.Fetch(context.Background(), resolved)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !strings.Contains(err.Error(), "missing path") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLocalFetchReturnsExpectedStructure(t *testing.T) {
	r := &LocalResolver{}
	resolved := &ResolvedSource{
		Name:  "test",
		Type:  "local",
		Path:  "./src/",
		Files: map[string]string{"a.md": "hash-a", "b.md": "hash-b"},
	}

	fetched, err := r.Fetch(context.Background(), resolved)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(fetched) != 2 {
		t.Fatalf("expected 2 files, got %d", len(fetched))
	}

	hashByPath := make(map[string]string)
	for _, f := range fetched {
		hashByPath[f.RelPath] = f.SHA256
	}
	if hashByPath["a.md"] != "hash-a" {
		t.Errorf("a.md hash = %q, want hash-a", hashByPath["a.md"])
	}
	if hashByPath["b.md"] != "hash-b" {
		t.Errorf("b.md hash = %q, want hash-b", hashByPath["b.md"])
	}
}

func TestLocalFetchWithRootSingleFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "policy.md"), []byte("# Policy\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r := &LocalResolver{}
	data, _ := os.ReadFile(filepath.Join(root, "policy.md"))
	hash := computeLocalHash(data)

	resolved := &ResolvedSource{
		Name:  "test",
		Type:  "local",
		Path:  "./policy.md",
		Files: map[string]string{"policy.md": hash},
	}

	fetched, err := r.FetchWithRoot(context.Background(), resolved, root)
	if err != nil {
		t.Fatalf("FetchWithRoot: %v", err)
	}
	if len(fetched) != 1 {
		t.Fatalf("expected 1 file, got %d", len(fetched))
	}
	if string(fetched[0].Content) != "# Policy\n" {
		t.Errorf("content = %q", string(fetched[0].Content))
	}
}

func TestLocalFetchWithRootNonexistentPath(t *testing.T) {
	r := &LocalResolver{}
	resolved := &ResolvedSource{
		Name:  "test",
		Type:  "local",
		Path:  "./nonexistent/",
		Files: map[string]string{"file.md": "hash"},
	}

	_, err := r.FetchWithRoot(context.Background(), resolved, t.TempDir())
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestLocalFetchWithRootMissingFile(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	r := &LocalResolver{}
	resolved := &ResolvedSource{
		Name:  "test",
		Type:  "local",
		Path:  "./src/",
		Files: map[string]string{"missing.md": "hash"},
	}

	_, err := r.FetchWithRoot(context.Background(), resolved, root)
	if err == nil {
		t.Fatal("expected error for missing file in directory")
	}
}

func TestLocalResolverCancelledContext(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "file.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := &LocalResolver{}
	src := config.Source{Name: "src", Type: "local", Path: "./src/"}

	_, err := r.Resolve(ctx, src, root)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

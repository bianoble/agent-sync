package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bianoble/agent-sync/internal/cache"
	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/source"
	"github.com/bianoble/agent-sync/internal/target"
)

func TestApplyTransforms(t *testing.T) {
	files := map[string][]byte{
		"rules.md": []byte("Hello {{.name}}! Org is {{.org}}."),
	}
	transforms := []config.Transform{
		{Source: "src", Type: "template", Vars: map[string]string{"name": "World"}},
	}
	globalVars := map[string]string{"org": "acme"}

	result, err := applyTransforms(files, transforms, globalVars)
	if err != nil {
		t.Fatalf("applyTransforms: %v", err)
	}

	got := string(result["rules.md"])
	if got != "Hello World! Org is acme." {
		t.Errorf("result = %q, want 'Hello World! Org is acme.'", got)
	}
}

func TestApplyTransformsSkipsNonTemplate(t *testing.T) {
	files := map[string][]byte{
		"file.md": []byte("original"),
	}
	transforms := []config.Transform{
		{Source: "src", Type: "custom", Command: "echo"},
	}

	result, err := applyTransforms(files, transforms, nil)
	if err != nil {
		t.Fatalf("applyTransforms: %v", err)
	}

	if string(result["file.md"]) != "original" {
		t.Error("custom transforms should be skipped (deferred)")
	}
}

func TestApplyTransformsBinarySkipped(t *testing.T) {
	// Binary content (contains null bytes) should pass through unchanged
	// since the template transform skips binary content.
	files := map[string][]byte{
		"binary.bin": {0x00, 0x01, 0x02},
	}
	transforms := []config.Transform{
		{Source: "src", Type: "template", Vars: map[string]string{"foo": "bar"}},
	}

	result, err := applyTransforms(files, transforms, nil)
	if err != nil {
		t.Fatalf("applyTransforms: %v", err)
	}

	got := result["binary.bin"]
	if len(got) != 3 || got[0] != 0x00 {
		t.Error("binary content should pass through unchanged")
	}
}

func TestRollback(t *testing.T) {
	projectRoot := t.TempDir()

	// Create a file that will be "modified" and needs rollback.
	destDir := filepath.Join(projectRoot, ".out")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := ".out/existing.md"
	absPath := filepath.Join(projectRoot, filePath)
	originalContent := []byte("original content")
	if err := os.WriteFile(absPath, originalContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file that was "newly written" (didn't exist before).
	newFilePath := ".out/new-file.md"
	newAbsPath := filepath.Join(projectRoot, newFilePath)
	if err := os.WriteFile(newAbsPath, []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}

	snapshots := []snapshot{
		{path: filePath, content: originalContent, existed: true},
		{path: newFilePath, existed: false},
	}

	writtenPaths := []string{filePath, newFilePath}

	rollback(projectRoot, writtenPaths, snapshots)

	// existing.md should be restored to original content.
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading restored file: %v", err)
	}
	if string(content) != string(originalContent) {
		t.Errorf("restored content = %q, want %q", string(content), string(originalContent))
	}

	// new-file.md should be removed.
	if _, err := os.Stat(newAbsPath); !os.IsNotExist(err) {
		t.Error("new file should have been removed during rollback")
	}
}

func TestSyncEngineWithTransforms(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	templateContent := []byte("Hello {{.name}}!")
	contentHash := cache.ComputeHash(templateContent)

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			files: []source.FetchedFile{
				{RelPath: "greeting.md", Content: templateContent, SHA256: contentHash},
			},
		},
	})

	eng := &SyncEngine{
		Registry:    reg,
		Cache:       c,
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version:   1,
		Variables: map[string]string{"name": "World"},
		Sources:   []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
		Targets:   []config.Target{{Source: "src", Destination: ".out/"}},
		Transforms: []config.Transform{
			{Source: "src", Type: "template"},
		},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Path:  "./src/",
				Files: map[string]lock.FileHash{"greeting.md": {SHA256: contentHash}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Sync(context.Background(), lf, cfg, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(result.Written) != 1 {
		t.Fatalf("written = %d, want 1", len(result.Written))
	}

	content, readErr := os.ReadFile(filepath.Join(projectRoot, ".out/greeting.md"))
	if readErr != nil {
		t.Fatalf("reading: %v", readErr)
	}
	if string(content) != "Hello World!" {
		t.Errorf("content = %q, want 'Hello World!'", string(content))
	}
}

func TestSyncEngineWithOverrides(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	content := []byte("base content")
	contentHash := cache.ComputeHash(content)

	// Create override file.
	if err := os.WriteFile(filepath.Join(projectRoot, "footer.md"), []byte("-- footer --"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			files: []source.FetchedFile{
				{RelPath: "rules.md", Content: content, SHA256: contentHash},
			},
		},
	})

	eng := &SyncEngine{
		Registry:    reg,
		Cache:       c,
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
		Targets: []config.Target{{Source: "src", Destination: ".out/"}},
		Overrides: []config.Override{
			{Target: "rules.md", Strategy: "append", File: "footer.md"},
		},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Path:  "./src/",
				Files: map[string]lock.FileHash{"rules.md": {SHA256: contentHash}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Sync(context.Background(), lf, cfg, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(result.Written) != 1 {
		t.Fatalf("written = %d, want 1", len(result.Written))
	}

	written, readErr := os.ReadFile(filepath.Join(projectRoot, ".out/rules.md"))
	if readErr != nil {
		t.Fatalf("reading: %v", readErr)
	}
	want := "base content\n-- footer --"
	if string(written) != want {
		t.Errorf("content = %q, want %q", string(written), want)
	}
}

func TestSyncEngineFetchError(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {err: fmt.Errorf("fetch failed")},
	})

	eng := &SyncEngine{
		Registry:    reg,
		Cache:       c,
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
		Targets: []config.Target{{Source: "src", Destination: ".out/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Path:  "./src/",
				Files: map[string]lock.FileHash{"file.md": {SHA256: "hash"}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Sync(context.Background(), lf, cfg, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync should not return error for individual fetch failures: %v", err)
	}

	if len(result.Errors) != 1 {
		t.Errorf("errors = %d, want 1", len(result.Errors))
	}
}

func TestSyncEngineDryRunModified(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	// Pre-write a file with different content.
	destDir := filepath.Join(projectRoot, ".out")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "file.md"), []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	newContent := []byte("new content")
	contentHash := cache.ComputeHash(newContent)

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			files: []source.FetchedFile{
				{RelPath: "file.md", Content: newContent, SHA256: contentHash},
			},
		},
	})

	eng := &SyncEngine{
		Registry:    reg,
		Cache:       c,
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
		Targets: []config.Target{{Source: "src", Destination: ".out/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Path:  "./src/",
				Files: map[string]lock.FileHash{"file.md": {SHA256: contentHash}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Sync(context.Background(), lf, cfg, SyncOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(result.Written) != 1 {
		t.Fatalf("written = %d, want 1", len(result.Written))
	}
	if result.Written[0].Action != "modified" {
		t.Errorf("action = %q, want 'modified'", result.Written[0].Action)
	}
}

func TestSyncEngineDryRunUnchanged(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	content := []byte("same content")
	contentHash := cache.ComputeHash(content)

	// Pre-write the same content.
	destDir := filepath.Join(projectRoot, ".out")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "file.md"), content, 0644); err != nil {
		t.Fatal(err)
	}

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			files: []source.FetchedFile{
				{RelPath: "file.md", Content: content, SHA256: contentHash},
			},
		},
	})

	eng := &SyncEngine{
		Registry:    reg,
		Cache:       c,
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
		Targets: []config.Target{{Source: "src", Destination: ".out/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Path:  "./src/",
				Files: map[string]lock.FileHash{"file.md": {SHA256: contentHash}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Sync(context.Background(), lf, cfg, SyncOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(result.Skipped) != 1 {
		t.Fatalf("skipped = %d, want 1", len(result.Skipped))
	}
	if result.Skipped[0].Action != "unchanged" {
		t.Errorf("action = %q, want 'unchanged'", result.Skipped[0].Action)
	}
}

func TestSyncEngineModifiesExistingFile(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	// Pre-write with different content.
	destDir := filepath.Join(projectRoot, ".out")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "file.md"), []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	newContent := []byte("new content")
	contentHash := cache.ComputeHash(newContent)

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			files: []source.FetchedFile{
				{RelPath: "file.md", Content: newContent, SHA256: contentHash},
			},
		},
	})

	eng := &SyncEngine{
		Registry:    reg,
		Cache:       c,
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
		Targets: []config.Target{{Source: "src", Destination: ".out/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Path:  "./src/",
				Files: map[string]lock.FileHash{"file.md": {SHA256: contentHash}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Sync(context.Background(), lf, cfg, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(result.Written) != 1 {
		t.Fatalf("written = %d, want 1", len(result.Written))
	}
	if result.Written[0].Action != "modified" {
		t.Errorf("action = %q, want 'modified'", result.Written[0].Action)
	}
}

func TestSyncEngineNoTargets(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {},
	})

	eng := &SyncEngine{
		Registry:    reg,
		Cache:       c,
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	// Source exists in lockfile but has no targets in config.
	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
	}

	contentHash := cache.ComputeHash([]byte("content"))
	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Path:  "./src/",
				Files: map[string]lock.FileHash{"file.md": {SHA256: contentHash}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Sync(context.Background(), lf, cfg, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(result.Written) != 0 {
		t.Errorf("written = %d, want 0 (no targets)", len(result.Written))
	}
}

func TestFetchSourceFilesFromCache(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	content := []byte("cached content")
	contentHash := cache.ComputeHash(content)

	// Pre-populate cache.
	if err := c.Put(contentHash, content); err != nil {
		t.Fatal(err)
	}

	// Mock resolver that would fail if called — content should come from cache.
	reg := newTestRegistry(map[string]*mockResolver{
		"local": {err: fmt.Errorf("should not be called")},
	})

	eng := &SyncEngine{
		Registry:    reg,
		Cache:       c,
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	ls := lock.LockedSource{
		Name: "src", Type: "local",
		Resolved: lock.ResolvedState{
			Path:  "./src/",
			Files: map[string]lock.FileHash{"file.md": {SHA256: contentHash}},
		},
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
	}

	files, err := eng.fetchSourceFiles(context.Background(), ls, cfg)
	if err != nil {
		t.Fatalf("fetchSourceFiles: %v", err)
	}

	if string(files["file.md"]) != "cached content" {
		t.Errorf("content = %q, want 'cached content'", string(files["file.md"]))
	}
}

func TestResolveAllTargetsError(t *testing.T) {
	// Use a tool map with a target that references an unknown tool.
	tm := target.NewToolMap(nil)
	cfg := config.Config{
		Targets: []config.Target{
			{Source: "src", Tools: []string{"nonexistent-tool"}},
		},
	}

	_, err := resolveAllTargets(tm, cfg)
	if err == nil {
		t.Error("expected error for unknown tool in target")
	}
}

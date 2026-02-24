package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bianoble/agent-sync/internal/cache"
	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/source"
	"github.com/bianoble/agent-sync/internal/target"
)

// mockResolver is a test resolver that returns predefined content.
type mockResolver struct {
	resolved *source.ResolvedSource
	files    []source.FetchedFile
	err      error
}

func (m *mockResolver) Resolve(ctx context.Context, src config.Source, projectRoot string) (*source.ResolvedSource, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.resolved, nil
}

func (m *mockResolver) Fetch(ctx context.Context, resolved *source.ResolvedSource) ([]source.FetchedFile, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.files, nil
}

func newTestRegistry(resolvers map[string]*mockResolver) *source.Registry {
	reg := source.NewRegistry()
	for typ, r := range resolvers {
		reg.Register(typ, r)
	}
	return reg
}

func TestSyncEngineBasic(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	contentHash := cache.ComputeHash([]byte("# Security Rules\n"))

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			files: []source.FetchedFile{
				{RelPath: "security.md", Content: []byte("# Security Rules\n"), SHA256: contentHash},
			},
		},
	})

	tm := target.NewToolMap(nil)

	eng := &SyncEngine{
		Registry:    reg,
		Cache:       c,
		ToolMap:     tm,
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{
			{Name: "rules", Type: "local", Path: "./rules/"},
		},
		Targets: []config.Target{
			{Source: "rules", Destination: ".custom/rules/"},
		},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{
			{
				Name: "rules",
				Type: "local",
				Resolved: lock.ResolvedState{
					Path: "./rules/",
					Files: map[string]lock.FileHash{
						"security.md": {SHA256: contentHash},
					},
				},
				Status: "ok",
			},
		},
	}

	result, err := eng.Sync(context.Background(), lf, cfg, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(result.Written) != 1 {
		t.Errorf("written = %d, want 1", len(result.Written))
	}

	// Verify file was written.
	content, readErr := os.ReadFile(filepath.Join(projectRoot, ".custom/rules/security.md"))
	if readErr != nil {
		t.Fatalf("reading synced file: %v", readErr)
	}
	if string(content) != "# Security Rules\n" {
		t.Errorf("content = %q", string(content))
	}
}

func TestSyncEngineDryRun(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	contentHash := cache.ComputeHash([]byte("content"))

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			files: []source.FetchedFile{
				{RelPath: "file.md", Content: []byte("content"), SHA256: contentHash},
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
		t.Fatalf("Sync dry-run: %v", err)
	}

	if len(result.Written) != 1 {
		t.Errorf("dry-run should report 1 new file, got %d", len(result.Written))
	}

	// File should NOT exist.
	absPath := filepath.Join(projectRoot, ".out/file.md")
	if _, err := os.Stat(absPath); !os.IsNotExist(err) {
		t.Error("dry-run should not write files")
	}
}

func TestSyncEngineSkipsUnchanged(t *testing.T) {
	projectRoot := t.TempDir()
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	content := []byte("existing content")
	contentHash := cache.ComputeHash(content)

	// Pre-write the file so it's already there.
	destDir := filepath.Join(projectRoot, ".out")
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(destDir, "file.md"), content, 0644)

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

	result, err := eng.Sync(context.Background(), lf, cfg, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(result.Skipped) != 1 {
		t.Errorf("should skip 1 unchanged file, got %d skipped", len(result.Skipped))
	}
	if len(result.Written) != 0 {
		t.Errorf("should write 0 files, got %d", len(result.Written))
	}
}

func TestCheckEngineClean(t *testing.T) {
	projectRoot := t.TempDir()
	content := []byte("# Rules\n")
	contentHash := sha256Hex(content)

	// Write the expected file.
	destDir := filepath.Join(projectRoot, ".custom/rules")
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(destDir, "security.md"), content, 0644)

	eng := &CheckEngine{
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "rules", Type: "local"}},
		Targets: []config.Target{{Source: "rules", Destination: ".custom/rules/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "rules", Type: "local",
			Resolved: lock.ResolvedState{
				Files: map[string]lock.FileHash{"security.md": {SHA256: contentHash}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Check(context.Background(), lf, cfg)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.Clean {
		t.Errorf("expected clean, got drifted=%v missing=%v", result.Drifted, result.Missing)
	}
}

func TestCheckEngineDrift(t *testing.T) {
	projectRoot := t.TempDir()

	// Write a file with different content.
	destDir := filepath.Join(projectRoot, ".out")
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(destDir, "file.md"), []byte("modified"), 0644)

	eng := &CheckEngine{
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local"}},
		Targets: []config.Target{{Source: "src", Destination: ".out/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Files: map[string]lock.FileHash{"file.md": {SHA256: "expected_hash"}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Check(context.Background(), lf, cfg)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Clean {
		t.Error("expected drift, got clean")
	}
	if len(result.Drifted) != 1 {
		t.Errorf("drifted = %d, want 1", len(result.Drifted))
	}
}

func TestCheckEngineMissing(t *testing.T) {
	projectRoot := t.TempDir()

	eng := &CheckEngine{
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local"}},
		Targets: []config.Target{{Source: "src", Destination: ".out/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Files: map[string]lock.FileHash{"missing.md": {SHA256: "abc"}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Check(context.Background(), lf, cfg)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Clean {
		t.Error("expected missing, got clean")
	}
	if len(result.Missing) != 1 {
		t.Errorf("missing = %d, want 1", len(result.Missing))
	}
}

func TestUpdateEngine(t *testing.T) {
	projectRoot := t.TempDir()
	contentHash := cache.ComputeHash([]byte("content"))

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			resolved: &source.ResolvedSource{
				Name:  "src",
				Type:  "local",
				Path:  "./src/",
				Files: map[string]string{"file.md": contentHash},
			},
			files: []source.FetchedFile{
				{RelPath: "file.md", Content: []byte("content"), SHA256: contentHash},
			},
		},
	})

	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	eng := &UpdateEngine{
		Registry:    reg,
		Cache:       c,
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
	}

	result, err := eng.Update(context.Background(), cfg, nil, UpdateOptions{})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if len(result.Updated) != 1 {
		t.Errorf("updated = %d, want 1", len(result.Updated))
	}
	if result.Lockfile == nil {
		t.Fatal("lockfile should not be nil")
	}
	if len(result.Lockfile.Sources) != 1 {
		t.Errorf("lockfile sources = %d, want 1", len(result.Lockfile.Sources))
	}
}

func TestUpdateEnginePartialNames(t *testing.T) {
	projectRoot := t.TempDir()
	contentHash := cache.ComputeHash([]byte("content"))

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			resolved: &source.ResolvedSource{
				Name:  "src-a",
				Type:  "local",
				Path:  "./a/",
				Files: map[string]string{"a.md": contentHash},
			},
			files: []source.FetchedFile{
				{RelPath: "a.md", Content: []byte("content"), SHA256: contentHash},
			},
		},
	})

	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)

	eng := &UpdateEngine{
		Registry:    reg,
		Cache:       c,
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{
			{Name: "src-a", Type: "local", Path: "./a/"},
			{Name: "src-b", Type: "local", Path: "./b/"},
		},
	}

	// Only update src-a.
	result, err := eng.Update(context.Background(), cfg, nil, UpdateOptions{SourceNames: []string{"src-a"}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if len(result.Updated) != 1 {
		t.Errorf("updated = %d, want 1", len(result.Updated))
	}
	if result.Updated[0].Name != "src-a" {
		t.Errorf("updated name = %q, want src-a", result.Updated[0].Name)
	}
}

func TestUpdateEngineDryRun(t *testing.T) {
	projectRoot := t.TempDir()
	contentHash := cache.ComputeHash([]byte("content"))

	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			resolved: &source.ResolvedSource{
				Name: "src", Type: "local", Path: "./src/",
				Files: map[string]string{"file.md": contentHash},
			},
		},
	})

	eng := &UpdateEngine{Registry: reg, ProjectRoot: projectRoot}
	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
	}

	result, err := eng.Update(context.Background(), cfg, nil, UpdateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Update dry-run: %v", err)
	}

	if result.Lockfile != nil {
		t.Error("dry-run should have nil lockfile")
	}
	if len(result.Updated) != 1 {
		t.Errorf("updated = %d, want 1", len(result.Updated))
	}
}

func TestStatusEngine(t *testing.T) {
	projectRoot := t.TempDir()
	content := []byte("# Rules\n")
	contentHash := sha256Hex(content)

	destDir := filepath.Join(projectRoot, ".out")
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(destDir, "file.md"), content, 0644)

	eng := &StatusEngine{
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local"}},
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

	statuses, err := eng.Status(context.Background(), lf, cfg, nil)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if len(statuses) != 1 {
		t.Fatalf("statuses = %d, want 1", len(statuses))
	}
	if statuses[0].State != "synced" {
		t.Errorf("state = %q, want synced", statuses[0].State)
	}
}

func TestStatusEnginePending(t *testing.T) {
	eng := &StatusEngine{
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: t.TempDir(),
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "new-src", Type: "local"}},
		Targets: []config.Target{{Source: "new-src", Destination: ".out/"}},
	}

	lf := lock.Lockfile{Version: 1} // empty

	statuses, err := eng.Status(context.Background(), lf, cfg, nil)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("statuses = %d", len(statuses))
	}
	if statuses[0].State != "pending" {
		t.Errorf("state = %q, want pending", statuses[0].State)
	}
}

func TestInfoEngine(t *testing.T) {
	cacheDir := t.TempDir()
	c, _ := cache.New(cacheDir)
	tm := target.NewToolMap(nil)

	info, err := Info("v0.1.0", nil, c, tm, "agent-sync.yaml", "agent-sync.lock")
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	if info.Version != "v0.1.0" {
		t.Errorf("version = %q", info.Version)
	}
	if info.SpecVersion != 1 {
		t.Errorf("spec = %d", info.SpecVersion)
	}
	if len(info.Tools) != 6 {
		t.Errorf("tools = %d, want 6", len(info.Tools))
	}
}

func TestPruneEngineDryRun(t *testing.T) {
	eng := &PruneEngine{
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: t.TempDir(),
	}

	cfg := config.Config{Version: 1}
	lf := lock.Lockfile{Version: 1}

	result, err := eng.Prune(context.Background(), lf, cfg, PruneOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	_ = result // dry run with no orphans should succeed
}

// Ensure unused import is consumed.
var _ = SyncResult{}

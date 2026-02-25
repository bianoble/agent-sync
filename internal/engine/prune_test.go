package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/target"
)

func TestPruneEngineRemovesOrphanedFiles(t *testing.T) {
	projectRoot := t.TempDir()
	tm := target.NewToolMap(nil)

	// Write a file for the orphaned source using a known tool path.
	cursorDir := filepath.Join(projectRoot, ".cursor/rules")
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "old-rule.md"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	eng := &PruneEngine{
		ToolMap:     tm,
		ProjectRoot: projectRoot,
	}

	// Config has no sources — the old source is orphaned.
	cfg := config.Config{Version: 1}

	// Lockfile has a source that's no longer in config.
	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "old-src", Type: "local",
			Resolved: lock.ResolvedState{
				Files: map[string]lock.FileHash{"old-rule.md": {SHA256: "hash"}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Prune(context.Background(), lf, cfg, PruneOptions{})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	if len(result.Removed) == 0 {
		t.Error("expected at least one removed file")
	}

	// Verify file was removed.
	if _, err := os.Stat(filepath.Join(cursorDir, "old-rule.md")); !os.IsNotExist(err) {
		t.Error("orphaned file should have been removed")
	}
}

func TestPruneEngineSkipsInConfigSources(t *testing.T) {
	projectRoot := t.TempDir()
	tm := target.NewToolMap(nil)

	eng := &PruneEngine{
		ToolMap:     tm,
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "active", Type: "local", Path: "./rules/"}},
		Targets: []config.Target{{Source: "active", Destination: ".out/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "active", Type: "local",
			Resolved: lock.ResolvedState{
				Files: map[string]lock.FileHash{"file.md": {SHA256: "hash"}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Prune(context.Background(), lf, cfg, PruneOptions{})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	if len(result.Removed) != 0 {
		t.Errorf("removed = %d, want 0 (source is still in config)", len(result.Removed))
	}
}

func TestPruneEngineDryRunNoRemoval(t *testing.T) {
	projectRoot := t.TempDir()
	tm := target.NewToolMap(nil)

	// Write a file for the orphaned source.
	cursorDir := filepath.Join(projectRoot, ".cursor/rules")
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "old-rule.md"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	eng := &PruneEngine{
		ToolMap:     tm,
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{Version: 1}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "old-src", Type: "local",
			Resolved: lock.ResolvedState{
				Files: map[string]lock.FileHash{"old-rule.md": {SHA256: "hash"}},
			},
			Status: "ok",
		}},
	}

	result, err := eng.Prune(context.Background(), lf, cfg, PruneOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	// Dry run should not remove anything.
	if len(result.Removed) != 0 {
		t.Errorf("dry-run removed = %d, want 0", len(result.Removed))
	}

	// File should still exist.
	if _, err := os.Stat(filepath.Join(cursorDir, "old-rule.md")); os.IsNotExist(err) {
		t.Error("dry-run should not remove files")
	}
}

func TestPruneEngineEmptyLockfile(t *testing.T) {
	eng := &PruneEngine{
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: t.TempDir(),
	}

	cfg := config.Config{Version: 1}
	lf := lock.Lockfile{Version: 1}

	result, err := eng.Prune(context.Background(), lf, cfg, PruneOptions{})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if len(result.Removed) != 0 {
		t.Errorf("removed = %d, want 0", len(result.Removed))
	}
}

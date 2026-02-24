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

func TestStatusEngineDrifted(t *testing.T) {
	projectRoot := t.TempDir()
	content := []byte("original content")
	contentHash := sha256Hex(content)

	destDir := filepath.Join(projectRoot, ".out")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write modified content.
	if err := os.WriteFile(filepath.Join(destDir, "file.md"), []byte("modified content"), 0644); err != nil {
		t.Fatal(err)
	}

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
	if statuses[0].State != "drifted" {
		t.Errorf("state = %q, want drifted", statuses[0].State)
	}
}

func TestStatusEngineMissing(t *testing.T) {
	projectRoot := t.TempDir()

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
				Files: map[string]lock.FileHash{"missing.md": {SHA256: "abc123"}},
			},
			Status: "ok",
		}},
	}

	statuses, err := eng.Status(context.Background(), lf, cfg, nil)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if statuses[0].State != "missing" {
		t.Errorf("state = %q, want missing", statuses[0].State)
	}
}

func TestSourceTypeFromConfigUnknown(t *testing.T) {
	cfg := config.Config{
		Sources: []config.Source{{Name: "known", Type: "git"}},
	}

	got := sourceTypeFromConfig("unknown", cfg)
	if got != "" {
		t.Errorf("expected empty for unknown source, got %q", got)
	}

	got = sourceTypeFromConfig("known", cfg)
	if got != "git" {
		t.Errorf("expected git, got %q", got)
	}
}

func TestComputeStateSynced(t *testing.T) {
	projectRoot := t.TempDir()
	content := []byte("test content")
	hash := sha256Hex(content)

	destDir := filepath.Join(projectRoot, ".out")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "file.md"), content, 0644); err != nil {
		t.Fatal(err)
	}

	ls := lock.LockedSource{
		Name: "src",
		Resolved: lock.ResolvedState{
			Files: map[string]lock.FileHash{"file.md": {SHA256: hash}},
		},
	}

	targets := []target.ResolvedTarget{{Destination: ".out/"}}
	state := computeState(projectRoot, ls, targets)
	if state != "synced" {
		t.Errorf("state = %q, want synced", state)
	}
}

func TestComputeStateNoTargets(t *testing.T) {
	projectRoot := t.TempDir()
	ls := lock.LockedSource{
		Name: "src",
		Resolved: lock.ResolvedState{
			Files: map[string]lock.FileHash{"file.md": {SHA256: "abc"}},
		},
	}

	state := computeState(projectRoot, ls, nil)
	if state != "synced" {
		t.Errorf("state = %q, want synced (no targets to check)", state)
	}
}

func TestComputeStateNoFiles(t *testing.T) {
	projectRoot := t.TempDir()
	ls := lock.LockedSource{
		Name:     "src",
		Resolved: lock.ResolvedState{},
	}

	targets := []target.ResolvedTarget{{Destination: ".out/"}}
	state := computeState(projectRoot, ls, targets)
	if state != "synced" {
		t.Errorf("state = %q, want synced (no files to check)", state)
	}
}

func TestStatusEngineNamedSources(t *testing.T) {
	projectRoot := t.TempDir()

	eng := &StatusEngine{
		ToolMap:     target.NewToolMap(nil),
		ProjectRoot: projectRoot,
	}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{
			{Name: "src-a", Type: "local"},
			{Name: "src-b", Type: "git"},
		},
		Targets: []config.Target{
			{Source: "src-a", Destination: ".out-a/"},
			{Source: "src-b", Destination: ".out-b/"},
		},
	}

	lf := lock.Lockfile{Version: 1}

	// Only query one source.
	statuses, err := eng.Status(context.Background(), lf, cfg, []string{"src-a"})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if len(statuses) != 1 {
		t.Fatalf("statuses = %d, want 1", len(statuses))
	}
	if statuses[0].Name != "src-a" {
		t.Errorf("name = %q, want src-a", statuses[0].Name)
	}
}

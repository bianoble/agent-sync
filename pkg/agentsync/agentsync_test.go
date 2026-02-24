package agentsync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bianoble/agent-sync/internal/lock"
)

// writeConfig writes a minimal valid config and returns its path.
func writeConfig(t *testing.T, dir string) string {
	t.Helper()
	cfgPath := filepath.Join(dir, "agent-sync.yaml")
	content := `version: 1
sources:
  - name: rules
    type: local
    path: ./rules/
targets:
  - source: rules
    destination: .out/
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

// writeLockfile writes a minimal valid lockfile and returns its path.
func writeLockfile(t *testing.T, dir, fileHash string) string {
	t.Helper()
	lfPath := filepath.Join(dir, "agent-sync.lock")
	lf := &lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "rules",
			Type: "local",
			Resolved: lock.ResolvedState{
				Path:  "./rules/",
				Files: map[string]lock.FileHash{"security.md": {SHA256: fileHash}},
			},
			Status: "ok",
		}},
	}
	if err := lock.Save(lfPath, lf); err != nil {
		t.Fatal(err)
	}
	return lfPath
}

func TestNewDefaultPaths(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	client, err := New(Options{
		ConfigPath:  cfgPath,
		ProjectRoot: dir,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.projectRoot != dir {
		t.Errorf("projectRoot = %q, want %q", client.projectRoot, dir)
	}
	if client.lockfilePath != "agent-sync.lock" {
		t.Errorf("lockfilePath = %q, want 'agent-sync.lock'", client.lockfilePath)
	}
}

func TestNewDefaultProjectRoot(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	client, err := New(Options{
		ConfigPath: cfgPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// ProjectRoot should be derived from config path's directory.
	if client.projectRoot != dir {
		t.Errorf("projectRoot = %q, want %q", client.projectRoot, dir)
	}
}

func TestNewWithCustomPaths(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	client, err := New(Options{
		ProjectRoot:      dir,
		ConfigPath:       cfgPath,
		LockfilePath:     filepath.Join(dir, "custom.lock"),
		CacheDir:         filepath.Join(dir, "cache"),
		SystemConfigPath: "/nonexistent/system.yaml",
		UserConfigPath:   "/nonexistent/user.yaml",
		NoInherit:        true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.noInherit != true {
		t.Error("noInherit should be true")
	}
	if client.systemConfigPath != "/nonexistent/system.yaml" {
		t.Errorf("systemConfigPath = %q", client.systemConfigPath)
	}
}

func TestClientSync(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	// Create source files.
	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("# Security Rules\n")
	if err := os.WriteFile(filepath.Join(rulesDir, "security.md"), content, 0644); err != nil {
		t.Fatal(err)
	}

	client, err := New(Options{
		ProjectRoot:  dir,
		ConfigPath:   cfgPath,
		LockfilePath: filepath.Join(dir, "agent-sync.lock"),
		NoInherit:    true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// First, update to create lockfile.
	updateResult, err := client.Update(context.Background(), UpdateOptions{})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updateResult.Updated) != 1 {
		t.Errorf("updated = %d, want 1", len(updateResult.Updated))
	}

	// Now sync.
	syncResult, err := client.Sync(context.Background(), SyncOptions{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(syncResult.Written) == 0 && len(syncResult.Skipped) == 0 {
		t.Error("expected at least one written or skipped file")
	}
}

func TestClientCheckDetectsDrift(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	// Create source files.
	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "security.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	client, err := New(Options{
		ProjectRoot:  dir,
		ConfigPath:   cfgPath,
		LockfilePath: filepath.Join(dir, "agent-sync.lock"),
		NoInherit:    true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Update to create lockfile.
	if _, err := client.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Write a target file with DIFFERENT content to simulate drift.
	outDir := filepath.Join(dir, ".out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "security.md"), []byte("drifted content"), 0644); err != nil {
		t.Fatal(err)
	}

	checkResult, err := client.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if checkResult.Clean {
		t.Error("expected drift, got clean")
	}
	if len(checkResult.Drifted) != 1 {
		t.Errorf("drifted = %d, want 1", len(checkResult.Drifted))
	}
}

func TestClientCheckDetectsMissing(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "security.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	client, err := New(Options{
		ProjectRoot:  dir,
		ConfigPath:   cfgPath,
		LockfilePath: filepath.Join(dir, "agent-sync.lock"),
		NoInherit:    true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Update to create lockfile, don't sync — files should be missing.
	if _, err := client.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	checkResult, err := client.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if checkResult.Clean {
		t.Error("expected missing files, got clean")
	}
	if len(checkResult.Missing) != 1 {
		t.Errorf("missing = %d, want 1", len(checkResult.Missing))
	}
}

func TestClientPrune(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "security.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	client, err := New(Options{
		ProjectRoot:  dir,
		ConfigPath:   cfgPath,
		LockfilePath: filepath.Join(dir, "agent-sync.lock"),
		NoInherit:    true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	pruneResult, err := client.Prune(context.Background(), PruneOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	_ = pruneResult // Should succeed with no errors.
}

func TestClientVerify(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "security.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	client, err := New(Options{
		ProjectRoot:  dir,
		ConfigPath:   cfgPath,
		LockfilePath: filepath.Join(dir, "agent-sync.lock"),
		NoInherit:    true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Update first to create lockfile.
	if _, err := client.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	verifyResult, err := client.Verify(context.Background(), nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(verifyResult.UpToDate) != 1 {
		t.Errorf("up-to-date = %d, want 1", len(verifyResult.UpToDate))
	}
}

func TestClientUpdateDryRun(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)
	lfPath := filepath.Join(dir, "agent-sync.lock")

	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "security.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	client, err := New(Options{
		ProjectRoot:  dir,
		ConfigPath:   cfgPath,
		LockfilePath: lfPath,
		NoInherit:    true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	result, err := client.Update(context.Background(), UpdateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Update dry-run: %v", err)
	}
	if len(result.Updated) != 1 {
		t.Errorf("updated = %d, want 1", len(result.Updated))
	}
	if result.Updated[0].Before != "(new)" {
		t.Errorf("before = %q, want '(new)'", result.Updated[0].Before)
	}

	// Lockfile should NOT be created.
	if _, err := os.Stat(lfPath); !os.IsNotExist(err) {
		t.Error("dry-run should not create lockfile")
	}
}

func TestClientUpdatePartialNames(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "agent-sync.yaml")
	cfgContent := `version: 1
sources:
  - name: src-a
    type: local
    path: ./a/
  - name: src-b
    type: local
    path: ./b/
targets:
  - source: src-a
    destination: .out-a/
  - source: src-b
    destination: .out-b/
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create source dirs.
	for _, name := range []string{"a", "b"} {
		d := filepath.Join(dir, name)
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "file.md"), []byte("content-"+name), 0644); err != nil {
			t.Fatal(err)
		}
	}

	client, err := New(Options{
		ProjectRoot:  dir,
		ConfigPath:   cfgPath,
		LockfilePath: filepath.Join(dir, "agent-sync.lock"),
		NoInherit:    true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	result, err := client.Update(context.Background(), UpdateOptions{
		SourceNames: []string{"src-a"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(result.Updated) != 1 {
		t.Errorf("updated = %d, want 1", len(result.Updated))
	}
	if result.Updated[0].Name != "src-a" {
		t.Errorf("name = %q, want 'src-a'", result.Updated[0].Name)
	}
}

func TestSummarizeLocked(t *testing.T) {
	tests := []struct {
		name string
		ls   lock.LockedSource
		want string
	}{
		{
			name: "git commit",
			ls: lock.LockedSource{
				Type:     "git",
				Resolved: lock.ResolvedState{Commit: "abcdef1234567890"},
			},
			want: "abcdef12",
		},
		{
			name: "url sha256",
			ls: lock.LockedSource{
				Type:     "url",
				Resolved: lock.ResolvedState{SHA256: "abcdef1234567890"},
			},
			want: "sha256:abcdef12",
		},
		{
			name: "local files",
			ls: lock.LockedSource{
				Type: "local",
				Resolved: lock.ResolvedState{
					Files: map[string]lock.FileHash{"a.md": {SHA256: "h1"}, "b.md": {SHA256: "h2"}},
				},
			},
			want: "(2 files)",
		},
		{
			name: "unknown",
			ls:   lock.LockedSource{Type: "custom"},
			want: "(unknown)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeLocked(tt.ls)
			if got != tt.want {
				t.Errorf("summarizeLocked() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientSyncDryRun(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "security.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	client, err := New(Options{
		ProjectRoot:  dir,
		ConfigPath:   cfgPath,
		LockfilePath: filepath.Join(dir, "agent-sync.lock"),
		NoInherit:    true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Update to create lockfile.
	if _, err := client.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Sync in dry-run mode.
	result, err := client.Sync(context.Background(), SyncOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	_ = result // Should succeed.
}

func TestToolMap(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "security.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	client, err := New(Options{
		ProjectRoot:  dir,
		ConfigPath:   cfgPath,
		LockfilePath: filepath.Join(dir, "agent-sync.lock"),
		NoInherit:    true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cfg, err := client.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	tm := client.toolMap(cfg)
	if tm == nil {
		t.Fatal("expected non-nil toolMap")
	}
}

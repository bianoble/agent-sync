package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/source"
)

func TestVerifyEngineAllUpToDate(t *testing.T) {
	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			resolved: &source.ResolvedSource{
				Name:  "src",
				Type:  "local",
				Path:  "./src/",
				Files: map[string]string{"file.md": "abc123"},
			},
		},
	})

	eng := &VerifyEngine{Registry: reg, ProjectRoot: t.TempDir()}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Path:  "./src/",
				Files: map[string]lock.FileHash{"file.md": {SHA256: "abc123"}},
			},
		}},
	}

	result, err := eng.Verify(context.Background(), lf, cfg, nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(result.UpToDate) != 1 {
		t.Errorf("up-to-date = %d, want 1", len(result.UpToDate))
	}
	if len(result.Changed) != 0 {
		t.Errorf("changed = %d, want 0", len(result.Changed))
	}
}

func TestVerifyEngineChanged(t *testing.T) {
	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			resolved: &source.ResolvedSource{
				Name:  "src",
				Type:  "local",
				Path:  "./src/",
				Files: map[string]string{"file.md": "newhash"},
			},
		},
	})

	eng := &VerifyEngine{Registry: reg, ProjectRoot: t.TempDir()}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Path:  "./src/",
				Files: map[string]lock.FileHash{"file.md": {SHA256: "oldhash"}},
			},
		}},
	}

	result, err := eng.Verify(context.Background(), lf, cfg, nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(result.Changed) != 1 {
		t.Errorf("changed = %d, want 1", len(result.Changed))
	}
	if len(result.UpToDate) != 0 {
		t.Errorf("up-to-date = %d, want 0", len(result.UpToDate))
	}
}

func TestVerifyEngineNotLocked(t *testing.T) {
	reg := newTestRegistry(map[string]*mockResolver{
		"local": {},
	})

	eng := &VerifyEngine{Registry: reg, ProjectRoot: t.TempDir()}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
	}

	lf := lock.Lockfile{Version: 1} // empty

	result, err := eng.Verify(context.Background(), lf, cfg, nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(result.Changed) != 1 {
		t.Errorf("changed = %d, want 1", len(result.Changed))
	}
	if result.Changed[0].Before != "(not locked)" {
		t.Errorf("before = %q, want '(not locked)'", result.Changed[0].Before)
	}
}

func TestVerifyEngineSourceNotInConfig(t *testing.T) {
	reg := newTestRegistry(map[string]*mockResolver{})

	eng := &VerifyEngine{Registry: reg, ProjectRoot: t.TempDir()}

	cfg := config.Config{Version: 1}
	lf := lock.Lockfile{Version: 1}

	result, err := eng.Verify(context.Background(), lf, cfg, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(result.Errors) != 1 {
		t.Errorf("errors = %d, want 1", len(result.Errors))
	}
}

func TestVerifyEngineResolveError(t *testing.T) {
	reg := newTestRegistry(map[string]*mockResolver{
		"local": {err: fmt.Errorf("resolve failed")},
	})

	eng := &VerifyEngine{Registry: reg, ProjectRoot: t.TempDir()}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "local", Path: "./src/"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "local",
			Resolved: lock.ResolvedState{
				Path: "./src/",
			},
		}},
	}

	result, err := eng.Verify(context.Background(), lf, cfg, nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(result.Errors) != 1 {
		t.Errorf("errors = %d, want 1", len(result.Errors))
	}
}

func TestVerifyEngineUnknownType(t *testing.T) {
	reg := newTestRegistry(map[string]*mockResolver{})

	eng := &VerifyEngine{Registry: reg, ProjectRoot: t.TempDir()}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "src", Type: "unknown"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "src", Type: "unknown",
			Resolved: lock.ResolvedState{},
		}},
	}

	result, err := eng.Verify(context.Background(), lf, cfg, nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(result.Errors) != 1 {
		t.Errorf("errors = %d, want 1", len(result.Errors))
	}
}

func TestVerifyEnginePartialSourceNames(t *testing.T) {
	reg := newTestRegistry(map[string]*mockResolver{
		"local": {
			resolved: &source.ResolvedSource{
				Name:  "src-a",
				Type:  "local",
				Path:  "./a/",
				Files: map[string]string{"a.md": "hash-a"},
			},
		},
	})

	eng := &VerifyEngine{Registry: reg, ProjectRoot: t.TempDir()}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{
			{Name: "src-a", Type: "local", Path: "./a/"},
			{Name: "src-b", Type: "local", Path: "./b/"},
		},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{
			{
				Name: "src-a", Type: "local",
				Resolved: lock.ResolvedState{
					Path:  "./a/",
					Files: map[string]lock.FileHash{"a.md": {SHA256: "hash-a"}},
				},
			},
			{
				Name: "src-b", Type: "local",
				Resolved: lock.ResolvedState{
					Path:  "./b/",
					Files: map[string]lock.FileHash{"b.md": {SHA256: "hash-b"}},
				},
			},
		},
	}

	// Only verify src-a.
	result, err := eng.Verify(context.Background(), lf, cfg, []string{"src-a"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(result.UpToDate) != 1 {
		t.Errorf("up-to-date = %d, want 1", len(result.UpToDate))
	}
}

func TestVerifyEngineGitCommitChanged(t *testing.T) {
	reg := newTestRegistry(map[string]*mockResolver{
		"git": {
			resolved: &source.ResolvedSource{
				Name:   "rules",
				Type:   "git",
				Commit: "newcommitsha1234",
				Files:  map[string]string{"file.md": "hash"},
			},
		},
	})

	eng := &VerifyEngine{Registry: reg, ProjectRoot: t.TempDir()}

	cfg := config.Config{
		Version: 1,
		Sources: []config.Source{{Name: "rules", Type: "git", Repo: "https://github.com/test/repo.git", Ref: "main"}},
	}

	lf := lock.Lockfile{
		Version: 1,
		Sources: []lock.LockedSource{{
			Name: "rules", Type: "git",
			Resolved: lock.ResolvedState{
				Commit: "oldcommitsha5678",
				Files:  map[string]lock.FileHash{"file.md": {SHA256: "hash"}},
			},
		}},
	}

	result, err := eng.Verify(context.Background(), lf, cfg, nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(result.Changed) != 1 {
		t.Errorf("changed = %d, want 1", len(result.Changed))
	}
}

func TestHasChanged(t *testing.T) {
	tests := []struct {
		ls       lock.LockedSource
		resolved *source.ResolvedSource
		name     string
		want     bool
	}{
		{
			name: "git commit changed",
			ls:   lock.LockedSource{Type: "git", Resolved: lock.ResolvedState{Commit: "aaa"}},
			resolved: &source.ResolvedSource{
				Type:   "git",
				Commit: "bbb",
			},
			want: true,
		},
		{
			name: "git commit same",
			ls:   lock.LockedSource{Type: "git", Resolved: lock.ResolvedState{Commit: "aaa"}},
			resolved: &source.ResolvedSource{
				Type:   "git",
				Commit: "aaa",
			},
			want: false,
		},
		{
			name: "file count changed",
			ls: lock.LockedSource{
				Type: "local",
				Resolved: lock.ResolvedState{
					Files: map[string]lock.FileHash{"a.md": {SHA256: "h1"}},
				},
			},
			resolved: &source.ResolvedSource{
				Type:  "local",
				Files: map[string]string{"a.md": "h1", "b.md": "h2"},
			},
			want: true,
		},
		{
			name: "file hash changed",
			ls: lock.LockedSource{
				Type: "local",
				Resolved: lock.ResolvedState{
					Files: map[string]lock.FileHash{"a.md": {SHA256: "old"}},
				},
			},
			resolved: &source.ResolvedSource{
				Type:  "local",
				Files: map[string]string{"a.md": "new"},
			},
			want: true,
		},
		{
			name: "file removed from resolved",
			ls: lock.LockedSource{
				Type: "local",
				Resolved: lock.ResolvedState{
					Files: map[string]lock.FileHash{"a.md": {SHA256: "h1"}, "b.md": {SHA256: "h2"}},
				},
			},
			resolved: &source.ResolvedSource{
				Type:  "local",
				Files: map[string]string{"a.md": "h1"},
			},
			want: true,
		},
		{
			name: "files identical",
			ls: lock.LockedSource{
				Type: "local",
				Resolved: lock.ResolvedState{
					Files: map[string]lock.FileHash{"a.md": {SHA256: "h1"}},
				},
			},
			resolved: &source.ResolvedSource{
				Type:  "local",
				Files: map[string]string{"a.md": "h1"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasChanged(tt.ls, tt.resolved)
			if got != tt.want {
				t.Errorf("hasChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummarizeLocked(t *testing.T) {
	tests := []struct {
		name string
		ls   lock.LockedSource
		want string
	}{
		{
			name: "git with long commit",
			ls:   lock.LockedSource{Type: "git", Resolved: lock.ResolvedState{Commit: "abcdef1234567890"}},
			want: "abcdef12",
		},
		{
			name: "git with short commit",
			ls:   lock.LockedSource{Type: "git", Resolved: lock.ResolvedState{Commit: "abc"}},
			want: "abc",
		},
		{
			name: "url with sha256",
			ls:   lock.LockedSource{Type: "url", Resolved: lock.ResolvedState{SHA256: "abcdef1234567890"}},
			want: "sha256:abcdef12",
		},
		{
			name: "url with short sha256",
			ls:   lock.LockedSource{Type: "url", Resolved: lock.ResolvedState{SHA256: "abc"}},
			want: "sha256:abc",
		},
		{
			name: "local with files",
			ls: lock.LockedSource{
				Type: "local",
				Resolved: lock.ResolvedState{
					Files: map[string]lock.FileHash{"a.md": {SHA256: "h1"}, "b.md": {SHA256: "h2"}},
				},
			},
			want: "(2 files)",
		},
		{
			name: "unknown type",
			ls:   lock.LockedSource{Type: "custom"},
			want: "(unknown)",
		},
		{
			name: "git with no commit",
			ls:   lock.LockedSource{Type: "git"},
			want: "(unknown)",
		},
		{
			name: "url with no sha256",
			ls:   lock.LockedSource{Type: "url"},
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

func TestSummarizeResolved(t *testing.T) {
	tests := []struct {
		name     string
		resolved *source.ResolvedSource
		want     string
	}{
		{
			name:     "git with long commit",
			resolved: &source.ResolvedSource{Type: "git", Commit: "abcdef1234567890"},
			want:     "abcdef12",
		},
		{
			name:     "git with short commit",
			resolved: &source.ResolvedSource{Type: "git", Commit: "abc"},
			want:     "abc",
		},
		{
			name:     "url with file hash",
			resolved: &source.ResolvedSource{Type: "url", Files: map[string]string{"file.md": "abcdef1234567890"}},
			want:     "sha256:abcdef12",
		},
		{
			name:     "local with files",
			resolved: &source.ResolvedSource{Type: "local", Files: map[string]string{"a.md": "h1", "b.md": "h2"}},
			want:     "(2 files)",
		},
		{
			name:     "unknown type",
			resolved: &source.ResolvedSource{Type: "custom"},
			want:     "(unknown)",
		},
		{
			name:     "git with no commit",
			resolved: &source.ResolvedSource{Type: "git"},
			want:     "(unknown)",
		},
		{
			name:     "url with no files",
			resolved: &source.ResolvedSource{Type: "url", Files: map[string]string{}},
			want:     "(unknown)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeResolved(tt.resolved)
			if got != tt.want {
				t.Errorf("summarizeResolved() = %q, want %q", got, tt.want)
			}
		})
	}
}

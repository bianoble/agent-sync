package source

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bianoble/agent-sync/internal/config"
)

func TestGitResolverMissingRepo(t *testing.T) {
	r := &GitResolver{}
	_, err := r.Resolve(context.Background(), config.Source{Name: "test", Type: "git", Ref: "main"}, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing repo")
	}
	if !strings.Contains(err.Error(), "repo is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGitResolverMissingRef(t *testing.T) {
	r := &GitResolver{}
	_, err := r.Resolve(context.Background(), config.Source{Name: "test", Type: "git", Repo: "https://example.com/repo.git"}, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing ref")
	}
	if !strings.Contains(err.Error(), "ref is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGitResolverWithLocalRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Create a bare repo with a file.
	bareRepo := t.TempDir()
	workDir := t.TempDir()

	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}

	// Init, add file, commit, create bare.
	run(workDir, "init", "-b", "main")
	if err := os.MkdirAll(filepath.Join(workDir, "core"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "core", "rules.md"), []byte("# Rules\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "other.md"), []byte("# Other\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run(workDir, "add", ".")
	run(workDir, "commit", "-m", "initial")
	run(workDir, "clone", "--bare", workDir, bareRepo)

	r := &GitResolver{}
	src := config.Source{
		Name:  "test-git",
		Type:  "git",
		Repo:  bareRepo,
		Ref:   "main",
		Paths: []string{"core/"},
	}

	resolved, err := r.Resolve(context.Background(), src, t.TempDir())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if resolved.Name != "test-git" {
		t.Errorf("name = %q", resolved.Name)
	}
	if resolved.Commit == "" {
		t.Error("commit should not be empty")
	}
	if resolved.Tree == "" {
		t.Error("tree should not be empty")
	}
	if len(resolved.Files) != 1 {
		t.Errorf("expected 1 file under core/, got %d: %v", len(resolved.Files), resolved.Files)
	}
	if _, ok := resolved.Files["core/rules.md"]; !ok {
		t.Errorf("expected core/rules.md in files, got: %v", resolved.Files)
	}

	// Fetch.
	resolved.Repo = bareRepo
	fetched, err := r.Fetch(context.Background(), resolved)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(fetched) != 1 {
		t.Fatalf("expected 1 fetched file, got %d", len(fetched))
	}
	if string(fetched[0].Content) != "# Rules\n" {
		t.Errorf("content = %q", string(fetched[0].Content))
	}
}

func TestGitResolverNoPathFilter(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	workDir := t.TempDir()
	bareRepo := t.TempDir()

	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}

	run(workDir, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(workDir, "a.md"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "b.md"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	run(workDir, "add", ".")
	run(workDir, "commit", "-m", "init")
	run(workDir, "clone", "--bare", workDir, bareRepo)

	r := &GitResolver{}
	src := config.Source{Name: "all", Type: "git", Repo: bareRepo, Ref: "main"}

	resolved, err := r.Resolve(context.Background(), src, t.TempDir())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolved.Files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(resolved.Files), resolved.Files)
	}
}

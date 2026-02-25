package transform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bianoble/agent-sync/internal/config"
)

func TestApplySingleAppend(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "footer.md"), []byte("-- footer --"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &OverrideProcessor{ProjectRoot: root}
	content := []byte("base content")
	ov := config.Override{Target: "file.md", Strategy: "append", File: "footer.md"}

	result, err := p.ApplySingle(content, ov)
	if err != nil {
		t.Fatalf("ApplySingle: %v", err)
	}

	got := string(result)
	if !strings.HasPrefix(got, "base content") {
		t.Errorf("should start with base content: %q", got)
	}
	if !strings.HasSuffix(got, "-- footer --") {
		t.Errorf("should end with footer: %q", got)
	}
}

func TestApplySinglePrepend(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "header.md"), []byte("-- header --"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &OverrideProcessor{ProjectRoot: root}
	content := []byte("base content")
	ov := config.Override{Target: "file.md", Strategy: "prepend", File: "header.md"}

	result, err := p.ApplySingle(content, ov)
	if err != nil {
		t.Fatalf("ApplySingle: %v", err)
	}

	got := string(result)
	if !strings.HasPrefix(got, "-- header --") {
		t.Errorf("should start with header: %q", got)
	}
	if !strings.HasSuffix(got, "base content") {
		t.Errorf("should end with base content: %q", got)
	}
}

func TestApplySingleReplace(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "replacement.md"), []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &OverrideProcessor{ProjectRoot: root}
	content := []byte("old content")
	ov := config.Override{Target: "file.md", Strategy: "replace", File: "replacement.md"}

	result, err := p.ApplySingle(content, ov)
	if err != nil {
		t.Fatalf("ApplySingle: %v", err)
	}

	if string(result) != "new content" {
		t.Errorf("result = %q, want 'new content'", string(result))
	}
}

func TestApplySingleInvalidStrategy(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &OverrideProcessor{ProjectRoot: root}
	ov := config.Override{Target: "file.md", Strategy: "merge", File: "file.md"}

	_, err := p.ApplySingle([]byte("content"), ov)
	if err == nil {
		t.Fatal("expected error for invalid strategy")
	}
	if !strings.Contains(err.Error(), "invalid strategy") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplySingleMissingFile(t *testing.T) {
	root := t.TempDir()
	p := &OverrideProcessor{ProjectRoot: root}
	ov := config.Override{Target: "file.md", Strategy: "append", File: "nonexistent.md"}

	_, err := p.ApplySingle([]byte("content"), ov)
	if err == nil {
		t.Fatal("expected error for missing override file")
	}
}

func TestApplyInvalidStrategy(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ov.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &OverrideProcessor{ProjectRoot: root}
	files := map[string][]byte{"file.md": []byte("base")}
	overrides := []config.Override{
		{Target: "file.md", Strategy: "invalid", File: "ov.md"},
	}

	_, err := p.Apply(files, overrides)
	if err == nil {
		t.Fatal("expected error for invalid strategy")
	}
}

func TestApplyMissingOverrideFile(t *testing.T) {
	root := t.TempDir()
	p := &OverrideProcessor{ProjectRoot: root}
	files := map[string][]byte{"file.md": []byte("base")}
	overrides := []config.Override{
		{Target: "file.md", Strategy: "append", File: "missing.md"},
	}

	_, err := p.Apply(files, overrides)
	if err == nil {
		t.Fatal("expected error for missing override file")
	}
}

func TestDetectConflictsThreeSources(t *testing.T) {
	dests := map[string][]string{
		".cursor/rules/security.md": {"source-a", "source-b", "source-c"},
	}
	err := DetectConflicts(dests, nil)
	if err == nil {
		t.Fatal("expected conflict error for 3 sources")
	}
	if !strings.Contains(err.Error(), "source-a") {
		t.Errorf("error should list source names: %v", err)
	}
}

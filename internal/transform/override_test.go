package transform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bianoble/agent-sync/internal/config"
)

func TestOverrideAppend(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "extra.md"), []byte("appended content"), 0644)

	p := &OverrideProcessor{ProjectRoot: root}
	files := map[string][]byte{
		"security.md": []byte("original"),
	}
	overrides := []config.Override{
		{Target: "security.md", Strategy: "append", File: "extra.md"},
	}

	result, err := p.Apply(files, overrides)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got := string(result["security.md"])
	if !strings.HasPrefix(got, "original") {
		t.Errorf("should start with original content: %q", got)
	}
	if !strings.HasSuffix(got, "appended content") {
		t.Errorf("should end with appended content: %q", got)
	}
}

func TestOverridePrepend(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "header.md"), []byte("prepended content"), 0644)

	p := &OverrideProcessor{ProjectRoot: root}
	files := map[string][]byte{
		"security.md": []byte("original"),
	}
	overrides := []config.Override{
		{Target: "security.md", Strategy: "prepend", File: "header.md"},
	}

	result, err := p.Apply(files, overrides)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got := string(result["security.md"])
	if !strings.HasPrefix(got, "prepended content") {
		t.Errorf("should start with prepended content: %q", got)
	}
	if !strings.HasSuffix(got, "original") {
		t.Errorf("should end with original content: %q", got)
	}
}

func TestOverrideReplace(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "replacement.md"), []byte("completely new"), 0644)

	p := &OverrideProcessor{ProjectRoot: root}
	files := map[string][]byte{
		"security.md": []byte("original"),
	}
	overrides := []config.Override{
		{Target: "security.md", Strategy: "replace", File: "replacement.md"},
	}

	result, err := p.Apply(files, overrides)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if string(result["security.md"]) != "completely new" {
		t.Errorf("got %q, want %q", string(result["security.md"]), "completely new")
	}
}

func TestOverrideTargetNotFound(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "extra.md"), []byte("content"), 0644)

	p := &OverrideProcessor{ProjectRoot: root}
	files := map[string][]byte{} // no synced files
	overrides := []config.Override{
		{Target: "missing.md", Strategy: "append", File: "extra.md"},
	}

	_, err := p.Apply(files, overrides)
	if err == nil {
		t.Fatal("expected error for missing target file")
	}
	if !strings.Contains(err.Error(), "does not exist after sync") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateOverridesFileMissing(t *testing.T) {
	root := t.TempDir()
	p := &OverrideProcessor{ProjectRoot: root}

	overrides := []config.Override{
		{Target: "file.md", Strategy: "append", File: "nonexistent.md"},
	}

	err := p.ValidateOverrides(overrides)
	if err == nil {
		t.Fatal("expected error for missing override file")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateOverridesFileExists(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "override.md"), []byte("content"), 0644)

	p := &OverrideProcessor{ProjectRoot: root}
	overrides := []config.Override{
		{Target: "file.md", Strategy: "append", File: "override.md"},
	}

	if err := p.ValidateOverrides(overrides); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetectConflictsNoConflict(t *testing.T) {
	dests := map[string][]string{
		".cursor/rules/security.md": {"rules"},
		".claude/security.md":       {"rules"},
	}
	if err := DetectConflicts(dests, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetectConflictsWithConflict(t *testing.T) {
	dests := map[string][]string{
		".cursor/rules/security.md": {"source-a", "source-b"},
	}
	err := DetectConflicts(dests, nil)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDetectConflictsWithOverride(t *testing.T) {
	dests := map[string][]string{
		".cursor/rules/security.md": {"source-a", "source-b"},
	}
	overrides := []config.Override{
		{Target: "security.md", Strategy: "append", File: "ext.md"},
	}
	if err := DetectConflicts(dests, overrides); err != nil {
		t.Fatalf("conflict should be resolved by override: %v", err)
	}
}

func TestOverrideDoesNotMutateOriginal(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "extra.md"), []byte("added"), 0644)

	p := &OverrideProcessor{ProjectRoot: root}
	original := []byte("original content")
	files := map[string][]byte{
		"file.md": original,
	}
	overrides := []config.Override{
		{Target: "file.md", Strategy: "append", File: "extra.md"},
	}

	_, err := p.Apply(files, overrides)
	if err != nil {
		t.Fatal(err)
	}

	// Original should be unchanged.
	if string(files["file.md"]) != "original content" {
		t.Errorf("original files map was mutated: %q", string(files["file.md"]))
	}
}

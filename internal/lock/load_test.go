package lock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadValidLockfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-sync.lock")
	if err := os.WriteFile(path, []byte(specExampleLockfile), 0644); err != nil {
		t.Fatal(err)
	}

	lf, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if lf.Version != 1 {
		t.Errorf("version = %d, want 1", lf.Version)
	}
	if len(lf.Sources) != 3 {
		t.Errorf("sources = %d, want 3", len(lf.Sources))
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/agent-sync.lock")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-sync.lock")

	original := &Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{
				Name: "test",
				Type: "url",
				Resolved: ResolvedState{
					URL:    "https://example.com/file.md",
					SHA256: "abc123",
				},
				Status: "ok",
			},
		},
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}

	if loaded.Version != 1 {
		t.Errorf("version = %d, want 1", loaded.Version)
	}
	if len(loaded.Sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(loaded.Sources))
	}
	if loaded.Sources[0].Name != "test" {
		t.Errorf("name = %q, want %q", loaded.Sources[0].Name, "test")
	}
	if loaded.Sources[0].Resolved.SHA256 != "abc123" {
		t.Errorf("sha256 = %q, want %q", loaded.Sources[0].Resolved.SHA256, "abc123")
	}

	// Verify temp file was cleaned up.
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file %s should not exist after save", tmpPath)
	}
}

func TestSaveAtomicity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-sync.lock")

	// Write initial content.
	initial := &Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{Name: "first", Type: "local", Resolved: ResolvedState{Path: "./a/"}, Status: "ok"},
		},
	}
	if err := Save(path, initial); err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	// Read it back to confirm.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var check Lockfile
	if err := yaml.Unmarshal(data, &check); err != nil {
		t.Fatal(err)
	}
	if check.Sources[0].Name != "first" {
		t.Errorf("initial name = %q, want %q", check.Sources[0].Name, "first")
	}

	// Overwrite with new content.
	updated := &Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{Name: "second", Type: "local", Resolved: ResolvedState{Path: "./b/"}, Status: "ok"},
		},
	}
	if err := Save(path, updated); err != nil {
		t.Fatalf("updated Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after update: %v", err)
	}
	if loaded.Sources[0].Name != "second" {
		t.Errorf("updated name = %q, want %q", loaded.Sources[0].Name, "second")
	}
}

func TestValidateVersionInvalid(t *testing.T) {
	lf := &Lockfile{Version: 99, Sources: []LockedSource{{Name: "s", Type: "local", Status: "ok"}}}
	errs := Validate(lf)
	if !containsSubstring(errs, "unsupported version") {
		t.Errorf("expected version error, got: %v", errs)
	}
}

func TestValidateDuplicateNames(t *testing.T) {
	lf := &Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{Name: "dup", Type: "local", Status: "ok"},
			{Name: "dup", Type: "local", Status: "ok"},
		},
	}
	errs := Validate(lf)
	if !containsSubstring(errs, "duplicate source name") {
		t.Errorf("expected duplicate name error, got: %v", errs)
	}
}

func TestValidateMissingFields(t *testing.T) {
	lf := &Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{},
		},
	}
	errs := Validate(lf)
	if !containsSubstring(errs, "'name' is required") {
		t.Errorf("expected name error, got: %v", errs)
	}
	if !containsSubstring(errs, "'type' is required") {
		t.Errorf("expected type error, got: %v", errs)
	}
	if !containsSubstring(errs, "'status' is required") {
		t.Errorf("expected status error, got: %v", errs)
	}
}

func TestValidateValidLockfile(t *testing.T) {
	lf := &Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{Name: "a", Type: "git", Status: "ok"},
			{Name: "b", Type: "url", Status: "ok"},
		},
	}
	errs := Validate(lf)
	if len(errs) > 0 {
		t.Errorf("expected no errors for valid lockfile, got: %v", errs)
	}
}

func TestValidationErrorFormat(t *testing.T) {
	verr := &ValidationError{Errors: []string{"error one", "error two"}}
	msg := verr.Error()
	if !strings.Contains(msg, "error one") || !strings.Contains(msg, "error two") {
		t.Errorf("error message missing details: %s", msg)
	}
}

func containsSubstring(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}

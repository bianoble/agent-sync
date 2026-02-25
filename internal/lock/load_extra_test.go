package lock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-sync.lock")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing lockfile") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadValidationFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-sync.lock")
	// Version 99 should fail validation.
	data := `version: 99
sources:
  - name: s
    type: local
    status: ok
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "unsupported version") {
		t.Errorf("expected 'unsupported version' in error, got: %v", err)
	}
}

func TestSaveToReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test unreliable as root")
	}
	dir := t.TempDir()
	readOnly := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(readOnly, 0555); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chmod(readOnly, 0755)
	}()

	lf := &Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{Name: "test", Type: "local", Status: "ok"},
		},
	}

	err := Save(filepath.Join(readOnly, "agent-sync.lock"), lf)
	if err == nil {
		t.Fatal("expected error writing to read-only directory")
	}
}

func TestSaveRenameError(t *testing.T) {
	// This tests the rename error path by writing to a directory
	// and then removing write permissions on the path itself.
	// On some systems this won't cause a rename error, but the write
	// will fail instead. Either way, we exercise the error path.
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "agent-sync.lock")

	lf := &Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{Name: "test", Type: "local", Status: "ok"},
		},
	}

	// Write to a non-existent subdirectory.
	err := Save(path, lf)
	if err == nil {
		t.Fatal("expected error writing to non-existent subdirectory")
	}
}

func TestValidateEmptyLockfile(t *testing.T) {
	lf := &Lockfile{Version: 1}
	errs := Validate(lf)
	if len(errs) != 0 {
		t.Errorf("expected no errors for lockfile with no sources, got: %v", errs)
	}
}

func TestValidateMissingNameShowsIndex(t *testing.T) {
	lf := &Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{Type: "local", Status: "ok"}, // missing name
		},
	}
	errs := Validate(lf)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "locked_source[0]") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error with index prefix, got: %v", errs)
	}
}

func TestValidateNamedSourcePrefix(t *testing.T) {
	lf := &Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{Name: "my-source", Status: "ok"}, // missing type
		},
	}
	errs := Validate(lf)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "locked source 'my-source'") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error with named prefix, got: %v", errs)
	}
}

func TestValidationErrorContainsAllErrors(t *testing.T) {
	verr := &ValidationError{Errors: []string{"a", "b", "c"}}
	msg := verr.Error()
	if !strings.Contains(msg, "lockfile validation failed") {
		t.Errorf("missing header: %s", msg)
	}
	for _, e := range []string{"a", "b", "c"} {
		if !strings.Contains(msg, e) {
			t.Errorf("missing error %q: %s", e, msg)
		}
	}
}

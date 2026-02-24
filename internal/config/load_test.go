package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-sync.yaml")
	if err := os.WriteFile(path, []byte(specExampleConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if len(cfg.Sources) != 3 {
		t.Errorf("sources = %d, want 3", len(cfg.Sources))
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/agent-sync.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestValidateVersionInvalid(t *testing.T) {
	cfg := &Config{Version: 99, Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}}, Targets: []Target{{Source: "s", Destination: "./out/"}}}
	errs := Validate(cfg)
	if !containsSubstring(errs, "unsupported version") {
		t.Errorf("expected version error, got: %v", errs)
	}
}

func TestValidateVersionZero(t *testing.T) {
	cfg := &Config{Version: 0, Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}}, Targets: []Target{{Source: "s", Destination: "./out/"}}}
	errs := Validate(cfg)
	if !containsSubstring(errs, "unsupported version") {
		t.Errorf("expected version error, got: %v", errs)
	}
}

func TestValidateNoSources(t *testing.T) {
	cfg := &Config{Version: 1}
	errs := Validate(cfg)
	if !containsSubstring(errs, "at least one source") {
		t.Errorf("expected source requirement error, got: %v", errs)
	}
}

func TestValidateDuplicateSourceNames(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{
			{Name: "dup", Type: "local", Path: "./a/"},
			{Name: "dup", Type: "local", Path: "./b/"},
		},
		Targets: []Target{
			{Source: "dup", Destination: "./out/"},
		},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "duplicate source name") {
		t.Errorf("expected duplicate name error, got: %v", errs)
	}
}

func TestValidateSourceMissingName(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "", Destination: "./out/"}},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "'name' is required") {
		t.Errorf("expected name required error, got: %v", errs)
	}
}

func TestValidateSourceMissingType(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "'type' is required") {
		t.Errorf("expected type required error, got: %v", errs)
	}
}

func TestValidateSourceUnknownType(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "ftp"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "unknown source type 'ftp'") {
		t.Errorf("expected unknown type error, got: %v", errs)
	}
}

func TestValidateGitSourceMissingFields(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "git"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "requires 'repo'") {
		t.Errorf("expected repo error, got: %v", errs)
	}
	if !containsSubstring(errs, "requires 'ref'") {
		t.Errorf("expected ref error, got: %v", errs)
	}
}

func TestValidateURLSourceMissingFields(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "url"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "requires 'url'") {
		t.Errorf("expected url error, got: %v", errs)
	}
	if !containsSubstring(errs, "requires 'checksum'") {
		t.Errorf("expected checksum error, got: %v", errs)
	}
}

func TestValidateLocalSourceMissingPath(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "requires 'path'") {
		t.Errorf("expected path error, got: %v", errs)
	}
}

func TestValidateTargetMutualExclusive(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "s", Tools: []string{"cursor"}, Destination: "./out/"}},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "mutually exclusive") {
		t.Errorf("expected mutual exclusion error, got: %v", errs)
	}
}

func TestValidateTargetNeitherToolsNorDest(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "s"}},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "one of 'tools' or 'destination'") {
		t.Errorf("expected tools/destination required error, got: %v", errs)
	}
}

func TestValidateTargetUndefinedSource(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "nonexistent", Destination: "./out/"}},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "undefined source 'nonexistent'") {
		t.Errorf("expected undefined source error, got: %v", errs)
	}
}

func TestValidateOverrideMissingFields(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
		Overrides: []Override{
			{},
		},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "'target' is required") {
		t.Errorf("expected target required error, got: %v", errs)
	}
	if !containsSubstring(errs, "'strategy' is required") {
		t.Errorf("expected strategy required error, got: %v", errs)
	}
	if !containsSubstring(errs, "'file' is required") {
		t.Errorf("expected file required error, got: %v", errs)
	}
}

func TestValidateOverrideInvalidStrategy(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
		Overrides: []Override{
			{Target: "f.md", Strategy: "merge", File: "local/f.md"},
		},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "invalid strategy 'merge'") {
		t.Errorf("expected invalid strategy error, got: %v", errs)
	}
}

func TestValidateTransformUndefinedSource(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
		Transforms: []Transform{
			{Source: "missing", Type: "template"},
		},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "undefined source 'missing'") {
		t.Errorf("expected undefined source error, got: %v", errs)
	}
}

func TestValidateTransformInvalidType(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
		Transforms: []Transform{
			{Source: "s", Type: "unknown"},
		},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "invalid type 'unknown'") {
		t.Errorf("expected invalid type error, got: %v", errs)
	}
}

func TestValidateCustomTransformMissingCommand(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
		Transforms: []Transform{
			{Source: "s", Type: "custom"},
		},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "requires 'command'") {
		t.Errorf("expected command required error, got: %v", errs)
	}
}

func TestValidateValidConfig(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Variables: map[string]string{
			"project": "test",
		},
		Sources: []Source{
			{Name: "rules", Type: "git", Repo: "https://example.com/repo.git", Ref: "v1.0.0"},
			{Name: "policy", Type: "url", URL: "https://example.com/p.md", Checksum: "sha256:abc"},
			{Name: "local", Type: "local", Path: "./agents/"},
		},
		Targets: []Target{
			{Source: "rules", Tools: []string{"cursor", "claude-code"}},
			{Source: "policy", Tools: []string{"cursor"}},
			{Source: "local", Destination: ".custom/"},
		},
		Overrides: []Override{
			{Target: "security.md", Strategy: "append", File: "local/ext.md"},
		},
		Transforms: []Transform{
			{Source: "rules", Type: "template", Vars: map[string]string{"project": "{{ .project }}"}},
		},
	}
	errs := Validate(cfg)
	if len(errs) > 0 {
		t.Errorf("expected no errors for valid config, got: %v", errs)
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

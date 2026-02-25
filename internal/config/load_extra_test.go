package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvNoInherit(t *testing.T) {
	tests := []struct {
		value string
		name  string
		want  bool
	}{
		{"", "empty", false},
		{"1", "1", true},
		{"true", "true", true},
		{"TRUE", "TRUE", true},
		{"True", "True", true},
		{"false", "false", false},
		{"0", "0", false},
		{"yes", "yes", false},
		{"  true  ", "with_spaces", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AGENT_SYNC_NO_INHERIT", tt.value)
			if got := EnvNoInherit(); got != tt.want {
				t.Errorf("EnvNoInherit() with %q = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `version: 1
sources:
  - name: rules
    type: local
    path: ./rules/
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if len(cfg.Sources) != 1 {
		t.Errorf("sources = %d, want 1", len(cfg.Sources))
	}
}

func TestParseInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("invalid: [yaml: broken"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseMissingFile(t *testing.T) {
	_, err := Parse("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadHierarchicalNoInheritProjectOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-sync.yaml")
	content := `version: 1
sources:
  - name: rules
    type: local
    path: ./rules/
targets:
  - source: rules
    destination: .out/
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadHierarchical(HierarchicalOptions{
		ProjectPath: path,
		NoInherit:   true,
	})
	if err != nil {
		t.Fatalf("LoadHierarchical: %v", err)
	}

	if len(result.Layers) != 1 {
		t.Errorf("layers = %d, want 1 (project only)", len(result.Layers))
	}
	if result.Layers[0].Level != LevelProject {
		t.Errorf("layer level = %q, want 'project'", result.Layers[0].Level)
	}
}

func TestLoadHierarchicalMissingProjectConfig(t *testing.T) {
	_, err := LoadHierarchical(HierarchicalOptions{
		ProjectPath:      "/nonexistent/agent-sync.yaml",
		SystemConfigPath: "/nonexistent/system.yaml",
		UserConfigPath:   "/nonexistent/user.yaml",
	})
	if err == nil {
		t.Fatal("expected error for missing project config")
	}
}

func TestLoadHierarchicalSystemParseError(t *testing.T) {
	dir := t.TempDir()

	// Valid project config.
	projectPath := filepath.Join(dir, "project.yaml")
	projectContent := `version: 1
sources:
  - name: rules
    type: local
    path: ./rules/
targets:
  - source: rules
    destination: .out/
`
	if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Broken system config.
	systemPath := filepath.Join(dir, "system.yaml")
	if err := os.WriteFile(systemPath, []byte("invalid: [yaml: broken"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadHierarchical(HierarchicalOptions{
		ProjectPath:      projectPath,
		SystemConfigPath: systemPath,
		UserConfigPath:   "/nonexistent/user.yaml",
	})
	if err == nil {
		t.Fatal("expected error for broken system config")
	}
	if !strings.Contains(err.Error(), "parsing") || !strings.Contains(err.Error(), "system") {
		t.Errorf("error should mention system parse failure: %v", err)
	}
}

func TestLoadHierarchicalMergesSystemAndProject(t *testing.T) {
	dir := t.TempDir()

	systemPath := filepath.Join(dir, "system.yaml")
	systemContent := `version: 1
variables:
  org: acme
sources:
  - name: org-rules
    type: local
    path: ./org/
`
	if err := os.WriteFile(systemPath, []byte(systemContent), 0644); err != nil {
		t.Fatal(err)
	}

	projectPath := filepath.Join(dir, "project.yaml")
	projectContent := `version: 1
sources:
  - name: team-rules
    type: local
    path: ./team/
targets:
  - source: org-rules
    destination: .org/
  - source: team-rules
    destination: .team/
`
	if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadHierarchical(HierarchicalOptions{
		ProjectPath:      projectPath,
		SystemConfigPath: systemPath,
		UserConfigPath:   "/nonexistent/user.yaml",
	})
	if err != nil {
		t.Fatalf("LoadHierarchical: %v", err)
	}

	if result.Config.Variables["org"] != "acme" {
		t.Errorf("variable org = %q, want 'acme'", result.Config.Variables["org"])
	}
	if len(result.Config.Sources) != 2 {
		t.Errorf("sources = %d, want 2", len(result.Config.Sources))
	}
}

func TestLoadHierarchicalNoInheritInvalidProject(t *testing.T) {
	_, err := LoadHierarchical(HierarchicalOptions{
		ProjectPath: "/nonexistent/config.yaml",
		NoInherit:   true,
	})
	if err == nil {
		t.Fatal("expected error for missing project config with NoInherit")
	}
}

func TestValidateToolDefinitionMissingFields(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
		ToolDefinitions: []ToolDefinition{
			{},
		},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "'name' is required") {
		t.Errorf("expected name required error, got: %v", errs)
	}
	if !containsSubstring(errs, "'destination' is required") {
		t.Errorf("expected destination required error, got: %v", errs)
	}
}

func TestValidateTransformMissingSource(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
		Transforms: []Transform{
			{Type: "template"},
		},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "'source' is required") {
		t.Errorf("expected source required error, got: %v", errs)
	}
}

func TestValidateTransformMissingType(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Source: "s", Destination: "./out/"}},
		Transforms: []Transform{
			{Source: "s"},
		},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "'type' is required") {
		t.Errorf("expected type required error, got: %v", errs)
	}
}

func TestValidateTargetMissingSource(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
		Targets: []Target{{Destination: "./out/"}},
	}
	errs := Validate(cfg)
	if !containsSubstring(errs, "'source' is required") {
		t.Errorf("expected source required error, got: %v", errs)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("broken: [yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("error should mention parsing: %v", err)
	}
}

func TestDefaultSystemConfigPathContainsConfigDir(t *testing.T) {
	path := defaultSystemConfigPath()
	if path == "" {
		t.Error("expected non-empty system config path")
	}
	if !strings.Contains(path, configDirName) {
		t.Errorf("path should contain %q: %s", configDirName, path)
	}
	if !strings.Contains(path, configFileName) {
		t.Errorf("path should contain %q: %s", configFileName, path)
	}
}

func TestDefaultUserConfigPathContainsConfigDir(t *testing.T) {
	path := defaultUserConfigPath()
	if path == "" {
		t.Skip("os.UserConfigDir() not available on this system")
	}
	if !strings.Contains(path, configDirName) {
		t.Errorf("path should contain %q: %s", configDirName, path)
	}
}

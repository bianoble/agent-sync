package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeVariablesDeepMerge(t *testing.T) {
	base := &Config{
		Version:   1,
		Variables: map[string]string{"org": "acme", "env": "prod"},
	}
	overlay := &Config{
		Version:   1,
		Variables: map[string]string{"env": "staging", "author": "alice"},
	}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if merged.Variables["org"] != "acme" {
		t.Errorf("org = %q, want %q", merged.Variables["org"], "acme")
	}
	if merged.Variables["env"] != "staging" {
		t.Errorf("env = %q, want %q (overlay should win)", merged.Variables["env"], "staging")
	}
	if merged.Variables["author"] != "alice" {
		t.Errorf("author = %q, want %q", merged.Variables["author"], "alice")
	}
}

func TestMergeSourcesByName(t *testing.T) {
	base := &Config{
		Version: 1,
		Sources: []Source{
			{Name: "shared", Type: "git", Repo: "https://base/repo.git", Ref: "v1.0"},
			{Name: "base-only", Type: "local", Path: "./base/"},
		},
	}
	overlay := &Config{
		Version: 1,
		Sources: []Source{
			{Name: "shared", Type: "git", Repo: "https://overlay/repo.git", Ref: "v2.0"},
			{Name: "overlay-only", Type: "local", Path: "./overlay/"},
		},
	}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if len(merged.Sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(merged.Sources))
	}

	// base-only should be preserved.
	found := findSource(merged.Sources, "base-only")
	if found == nil {
		t.Error("base-only source missing from merged result")
	}

	// shared should come from overlay (full replacement).
	shared := findSource(merged.Sources, "shared")
	if shared == nil {
		t.Fatal("shared source missing from merged result")
	}
	if shared.Ref != "v2.0" {
		t.Errorf("shared.Ref = %q, want %q (overlay should replace)", shared.Ref, "v2.0")
	}
	if shared.Repo != "https://overlay/repo.git" {
		t.Errorf("shared.Repo = %q, want %q", shared.Repo, "https://overlay/repo.git")
	}

	// overlay-only should be present.
	if findSource(merged.Sources, "overlay-only") == nil {
		t.Error("overlay-only source missing from merged result")
	}
}

func TestMergeToolDefinitionsByName(t *testing.T) {
	base := &Config{
		Version: 1,
		ToolDefinitions: []ToolDefinition{
			{Name: "cursor", Destination: ".cursor/base/"},
			{Name: "custom-base", Destination: ".base/"},
		},
	}
	overlay := &Config{
		Version: 1,
		ToolDefinitions: []ToolDefinition{
			{Name: "cursor", Destination: ".cursor/override/"},
		},
	}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if len(merged.ToolDefinitions) != 2 {
		t.Fatalf("expected 2 tool defs, got %d", len(merged.ToolDefinitions))
	}

	for _, td := range merged.ToolDefinitions {
		if td.Name == "cursor" && td.Destination != ".cursor/override/" {
			t.Errorf("cursor destination = %q, want %q", td.Destination, ".cursor/override/")
		}
	}
}

func TestMergeTargetsConcatenate(t *testing.T) {
	base := &Config{
		Version: 1,
		Targets: []Target{{Source: "a", Destination: "./a/"}},
	}
	overlay := &Config{
		Version: 1,
		Targets: []Target{{Source: "b", Destination: "./b/"}},
	}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if len(merged.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(merged.Targets))
	}
	if merged.Targets[0].Source != "a" {
		t.Errorf("targets[0].Source = %q, want %q (base first)", merged.Targets[0].Source, "a")
	}
	if merged.Targets[1].Source != "b" {
		t.Errorf("targets[1].Source = %q, want %q (overlay second)", merged.Targets[1].Source, "b")
	}
}

func TestMergeOverridesConcatenate(t *testing.T) {
	base := &Config{
		Version:   1,
		Overrides: []Override{{Target: "a.md", Strategy: "append", File: "a-ext.md"}},
	}
	overlay := &Config{
		Version:   1,
		Overrides: []Override{{Target: "b.md", Strategy: "prepend", File: "b-ext.md"}},
	}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if len(merged.Overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(merged.Overrides))
	}
}

func TestMergeTransformsConcatenate(t *testing.T) {
	base := &Config{
		Version:    1,
		Transforms: []Transform{{Source: "a", Type: "template"}},
	}
	overlay := &Config{
		Version:    1,
		Transforms: []Transform{{Source: "b", Type: "custom", Command: "run.sh"}},
	}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if len(merged.Transforms) != 2 {
		t.Fatalf("expected 2 transforms, got %d", len(merged.Transforms))
	}
}

func TestMergeVersionMismatch(t *testing.T) {
	base := &Config{Version: 1}
	overlay := &Config{Version: 2}

	_, err := Merge(base, overlay)
	if err == nil {
		t.Fatal("expected version mismatch error")
	}
	if !strings.Contains(err.Error(), "version mismatch") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMergeVersionZeroInherits(t *testing.T) {
	base := &Config{Version: 1}
	overlay := &Config{Version: 0} // doesn't declare

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Version != 1 {
		t.Errorf("version = %d, want 1 (inherited from base)", merged.Version)
	}
}

func TestMergeNilBase(t *testing.T) {
	overlay := &Config{Version: 1}
	merged, err := Merge(nil, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Version != 1 {
		t.Errorf("version = %d, want 1", merged.Version)
	}
}

func TestMergeNilOverlay(t *testing.T) {
	base := &Config{Version: 1}
	merged, err := Merge(base, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Version != 1 {
		t.Errorf("version = %d, want 1", merged.Version)
	}
}

func TestMergeAllThreeLayers(t *testing.T) {
	system := &Config{
		Version:   1,
		Variables: map[string]string{"org": "acme", "env": "prod"},
		Sources:   []Source{{Name: "org-rules", Type: "git", Repo: "https://acme/rules.git", Ref: "v1.0"}},
		Targets:   []Target{{Source: "org-rules", Tools: []string{"cursor"}}},
	}
	user := &Config{
		Variables: map[string]string{"env": "staging", "author": "alice"},
	}
	project := &Config{
		Version:   1,
		Variables: map[string]string{"env": "dev", "project": "my-app"},
		Sources: []Source{
			{Name: "org-rules", Type: "git", Repo: "https://acme/rules.git", Ref: "v2.0"},
			{Name: "local-rules", Type: "local", Path: "./agents/"},
		},
		Targets: []Target{{Source: "local-rules", Destination: ".custom/"}},
	}

	merged, err := MergeAll([]*Config{system, user, project})
	if err != nil {
		t.Fatalf("MergeAll: %v", err)
	}

	// Variables: system org preserved, env=dev from project, author from user, project from project.
	if merged.Variables["org"] != "acme" {
		t.Errorf("org = %q, want %q", merged.Variables["org"], "acme")
	}
	if merged.Variables["env"] != "dev" {
		t.Errorf("env = %q, want %q", merged.Variables["env"], "dev")
	}
	if merged.Variables["author"] != "alice" {
		t.Errorf("author = %q, want %q", merged.Variables["author"], "alice")
	}
	if merged.Variables["project"] != "my-app" {
		t.Errorf("project = %q, want %q", merged.Variables["project"], "my-app")
	}

	// Sources: org-rules replaced by project (v2.0), local-rules added.
	if len(merged.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(merged.Sources))
	}
	orgRules := findSource(merged.Sources, "org-rules")
	if orgRules == nil || orgRules.Ref != "v2.0" {
		t.Error("org-rules should be v2.0 from project layer")
	}

	// Targets: concatenated (system first, then project).
	if len(merged.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(merged.Targets))
	}
}

func TestMergeAllEmpty(t *testing.T) {
	_, err := MergeAll(nil)
	if err == nil {
		t.Fatal("expected error for empty configs")
	}
}

func TestLoadHierarchicalNoInherit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-sync.yaml")
	if err := os.WriteFile(path, []byte(specExampleConfig), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadHierarchical(HierarchicalOptions{
		ProjectPath: path,
		NoInherit:   true,
	})
	if err != nil {
		t.Fatalf("LoadHierarchical: %v", err)
	}

	if result.Config.Version != 1 {
		t.Errorf("version = %d, want 1", result.Config.Version)
	}
	if len(result.Layers) != 1 {
		t.Errorf("expected 1 layer with NoInherit, got %d", len(result.Layers))
	}
	if result.Layers[0].Level != LevelProject {
		t.Errorf("layer.Level = %q, want %q", result.Layers[0].Level, LevelProject)
	}
}

func TestLoadHierarchicalMergesLayers(t *testing.T) {
	dir := t.TempDir()

	// System config: defines a source and variable.
	sysDir := filepath.Join(dir, "system")
	if err := os.MkdirAll(sysDir, 0755); err != nil {
		t.Fatal(err)
	}
	sysConfig := `
version: 1
variables:
  org: acme
sources:
  - name: org-policy
    type: url
    url: https://acme.com/policy.md
    checksum: sha256:abc
`
	if err := os.WriteFile(filepath.Join(sysDir, "agent-sync.yaml"), []byte(sysConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Project config: adds its own source and target for both.
	projConfig := `
version: 1
variables:
  project: my-app
sources:
  - name: local-rules
    type: local
    path: ./agents/
targets:
  - source: org-policy
    tools: [cursor]
  - source: local-rules
    destination: .custom/
`
	projPath := filepath.Join(dir, "agent-sync.yaml")
	if err := os.WriteFile(projPath, []byte(projConfig), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadHierarchical(HierarchicalOptions{
		ProjectPath:      projPath,
		SystemConfigPath: filepath.Join(sysDir, "agent-sync.yaml"),
		UserConfigPath:   filepath.Join(dir, "nonexistent", "agent-sync.yaml"), // skip
	})
	if err != nil {
		t.Fatalf("LoadHierarchical: %v", err)
	}

	// Should have 2 sources (org-policy from system, local-rules from project).
	if len(result.Config.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(result.Config.Sources))
	}

	// Variables merged.
	if result.Config.Variables["org"] != "acme" {
		t.Errorf("org = %q, want %q", result.Config.Variables["org"], "acme")
	}
	if result.Config.Variables["project"] != "my-app" {
		t.Errorf("project = %q, want %q", result.Config.Variables["project"], "my-app")
	}

	// Layers metadata: system loaded, user not found, project loaded.
	loadedCount := 0
	for _, l := range result.Layers {
		if l.Loaded {
			loadedCount++
		}
	}
	if loadedCount != 2 {
		t.Errorf("expected 2 loaded layers, got %d", loadedCount)
	}
}

func TestLoadHierarchicalVersionMismatch(t *testing.T) {
	dir := t.TempDir()

	sysConfig := `version: 2
sources:
  - name: s
    type: local
    path: ./a/
`
	sysPath := filepath.Join(dir, "system.yaml")
	if err := os.WriteFile(sysPath, []byte(sysConfig), 0644); err != nil {
		t.Fatal(err)
	}

	projConfig := `version: 1
sources:
  - name: p
    type: local
    path: ./b/
targets:
  - source: p
    destination: ./out/
`
	projPath := filepath.Join(dir, "project.yaml")
	if err := os.WriteFile(projPath, []byte(projConfig), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadHierarchical(HierarchicalOptions{
		ProjectPath:      projPath,
		SystemConfigPath: sysPath,
		UserConfigPath:   filepath.Join(dir, "nonexistent.yaml"),
	})
	if err == nil {
		t.Fatal("expected version mismatch error")
	}
	if !strings.Contains(err.Error(), "version mismatch") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadHierarchicalParseError(t *testing.T) {
	dir := t.TempDir()

	sysPath := filepath.Join(dir, "system.yaml")
	if err := os.WriteFile(sysPath, []byte("invalid: [yaml: broken"), 0644); err != nil {
		t.Fatal(err)
	}

	projPath := filepath.Join(dir, "project.yaml")
	if err := os.WriteFile(projPath, []byte(specExampleConfig), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadHierarchical(HierarchicalOptions{
		ProjectPath:      projPath,
		SystemConfigPath: sysPath,
		UserConfigPath:   filepath.Join(dir, "nonexistent.yaml"),
	})
	if err == nil {
		t.Fatal("expected parse error for invalid system config")
	}
}

func findSource(sources []Source, name string) *Source {
	for i := range sources {
		if sources[i].Name == name {
			return &sources[i]
		}
	}
	return nil
}

package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// specExampleConfig is the example from spec Section 3.1.
const specExampleConfig = `
version: 1

variables:
  project: my-project
  language: go

sources:

  - name: base-rules
    type: git
    repo: https://github.com/org/rules.git
    ref: v1.2.0
    paths:
      - core/

  - name: security-policy
    type: url
    url: https://example.com/policy.md
    checksum: sha256:abcdef

  - name: team-standards
    type: local
    path: ./agents/standards/

targets:

  - source: base-rules
    tools: [cursor, claude-code, copilot]

  - source: security-policy
    tools: [cursor, claude-code]

  - source: team-standards
    destination: .custom/agents/

overrides:

  - target: security.md
    strategy: append
    file: local/security-extension.md

transforms:

  - source: base-rules
    type: template
    vars:
      project: "{{ .project }}"

  - source: team-standards
    type: custom
    command: "./scripts/merge-yaml.sh"
    output_hash: sha256:expected
`

func TestConfigParseSpecExample(t *testing.T) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(specExampleConfig), &cfg); err != nil {
		t.Fatalf("failed to parse spec example config: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}

	if len(cfg.Variables) != 2 {
		t.Errorf("variables count = %d, want 2", len(cfg.Variables))
	}
	if cfg.Variables["project"] != "my-project" {
		t.Errorf("variables[project] = %q, want %q", cfg.Variables["project"], "my-project")
	}

	if len(cfg.Sources) != 3 {
		t.Fatalf("sources count = %d, want 3", len(cfg.Sources))
	}

	// Git source.
	git := cfg.Sources[0]
	if git.Name != "base-rules" {
		t.Errorf("sources[0].name = %q, want %q", git.Name, "base-rules")
	}
	if git.Type != "git" {
		t.Errorf("sources[0].type = %q, want %q", git.Type, "git")
	}
	if git.Repo != "https://github.com/org/rules.git" {
		t.Errorf("sources[0].repo = %q, want %q", git.Repo, "https://github.com/org/rules.git")
	}
	if git.Ref != "v1.2.0" {
		t.Errorf("sources[0].ref = %q, want %q", git.Ref, "v1.2.0")
	}
	if len(git.Paths) != 1 || git.Paths[0] != "core/" {
		t.Errorf("sources[0].paths = %v, want [core/]", git.Paths)
	}

	// URL source.
	url := cfg.Sources[1]
	if url.Name != "security-policy" {
		t.Errorf("sources[1].name = %q, want %q", url.Name, "security-policy")
	}
	if url.Type != "url" {
		t.Errorf("sources[1].type = %q, want %q", url.Type, "url")
	}
	if url.URL != "https://example.com/policy.md" {
		t.Errorf("sources[1].url = %q", url.URL)
	}
	if url.Checksum != "sha256:abcdef" {
		t.Errorf("sources[1].checksum = %q", url.Checksum)
	}

	// Local source.
	local := cfg.Sources[2]
	if local.Name != "team-standards" {
		t.Errorf("sources[2].name = %q", local.Name)
	}
	if local.Type != "local" {
		t.Errorf("sources[2].type = %q", local.Type)
	}
	if local.Path != "./agents/standards/" {
		t.Errorf("sources[2].path = %q", local.Path)
	}

	// Targets.
	if len(cfg.Targets) != 3 {
		t.Fatalf("targets count = %d, want 3", len(cfg.Targets))
	}
	if cfg.Targets[0].Source != "base-rules" {
		t.Errorf("targets[0].source = %q", cfg.Targets[0].Source)
	}
	if len(cfg.Targets[0].Tools) != 3 {
		t.Errorf("targets[0].tools count = %d, want 3", len(cfg.Targets[0].Tools))
	}
	if cfg.Targets[2].Destination != ".custom/agents/" {
		t.Errorf("targets[2].destination = %q", cfg.Targets[2].Destination)
	}

	// Overrides.
	if len(cfg.Overrides) != 1 {
		t.Fatalf("overrides count = %d, want 1", len(cfg.Overrides))
	}
	if cfg.Overrides[0].Strategy != "append" {
		t.Errorf("overrides[0].strategy = %q", cfg.Overrides[0].Strategy)
	}

	// Transforms.
	if len(cfg.Transforms) != 2 {
		t.Fatalf("transforms count = %d, want 2", len(cfg.Transforms))
	}
	if cfg.Transforms[0].Type != "template" {
		t.Errorf("transforms[0].type = %q", cfg.Transforms[0].Type)
	}
	if cfg.Transforms[1].Type != "custom" {
		t.Errorf("transforms[1].type = %q", cfg.Transforms[1].Type)
	}
	if cfg.Transforms[1].Command != "./scripts/merge-yaml.sh" {
		t.Errorf("transforms[1].command = %q", cfg.Transforms[1].Command)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	original := Config{
		Version: 1,
		Variables: map[string]string{
			"project": "test",
		},
		Sources: []Source{
			{Name: "src", Type: "git", Repo: "https://example.com/repo.git", Ref: "main"},
		},
		Targets: []Target{
			{Source: "src", Tools: []string{"cursor"}},
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var roundTripped Config
	if err := yaml.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if roundTripped.Version != original.Version {
		t.Errorf("version = %d, want %d", roundTripped.Version, original.Version)
	}
	if roundTripped.Variables["project"] != "test" {
		t.Errorf("variables[project] = %q, want %q", roundTripped.Variables["project"], "test")
	}
	if len(roundTripped.Sources) != 1 {
		t.Fatalf("sources count = %d, want 1", len(roundTripped.Sources))
	}
	if roundTripped.Sources[0].Name != "src" {
		t.Errorf("sources[0].name = %q", roundTripped.Sources[0].Name)
	}
}

func TestConfigUnknownFieldsIgnored(t *testing.T) {
	input := `
version: 1
unknown_field: should be ignored
sources:
  - name: test
    type: local
    path: ./test/
    future_field: also ignored
targets:
  - source: test
    destination: ./out/
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatalf("should ignore unknown fields: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
}

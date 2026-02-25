package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInitCreatesConfig(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "agent-sync.yaml")

	// Override the global configPath used by the init command.
	old := configPath
	configPath = outPath
	defer func() { configPath = old }()

	initForce = false
	err := initCmd.RunE(initCmd, nil)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("config file is empty")
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "agent-sync.yaml")

	if err := os.WriteFile(outPath, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	old := configPath
	configPath = outPath
	defer func() { configPath = old }()

	initForce = false
	err := initCmd.RunE(initCmd, nil)
	if err == nil {
		t.Fatal("expected error when file exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists': %v", err)
	}
}

func TestInitForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "agent-sync.yaml")

	if err := os.WriteFile(outPath, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	old := configPath
	configPath = outPath
	defer func() { configPath = old }()

	initForce = true
	err := initCmd.RunE(initCmd, nil)
	if err != nil {
		t.Fatalf("init --force: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "old content" {
		t.Error("file was not overwritten")
	}
}

func TestInitTemplateIsValidYAML(t *testing.T) {
	var out map[string]any
	if err := yaml.Unmarshal([]byte(initTemplate), &out); err != nil {
		t.Fatalf("template is not valid YAML: %v", err)
	}
	if out["version"] == nil {
		t.Error("template should contain 'version'")
	}
}

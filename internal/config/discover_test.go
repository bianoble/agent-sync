package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoverPathsAllLevels(t *testing.T) {
	layers := DiscoverPaths(DiscoverOptions{
		ProjectPath:      "./agent-sync.yaml",
		SystemConfigPath: "/etc/agent-sync/agent-sync.yaml",
		UserConfigPath:   "/home/user/.config/agent-sync/agent-sync.yaml",
	})

	if len(layers) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(layers))
	}

	if layers[0].Level != LevelSystem {
		t.Errorf("layers[0].Level = %q, want %q", layers[0].Level, LevelSystem)
	}
	if layers[1].Level != LevelUser {
		t.Errorf("layers[1].Level = %q, want %q", layers[1].Level, LevelUser)
	}
	if layers[2].Level != LevelProject {
		t.Errorf("layers[2].Level = %q, want %q", layers[2].Level, LevelProject)
	}
}

func TestDiscoverPathsDeduplication(t *testing.T) {
	// If system and project point to the same file, deduplicate.
	samePath, err := filepath.Abs("./agent-sync.yaml")
	if err != nil {
		t.Fatal(err)
	}

	layers := DiscoverPaths(DiscoverOptions{
		ProjectPath:      samePath,
		SystemConfigPath: samePath,
		UserConfigPath:   "/other/path/agent-sync.yaml",
	})

	// System gets added first, then user, then project is deduplicated.
	if len(layers) != 2 {
		t.Fatalf("expected 2 layers (deduped), got %d", len(layers))
	}
	if layers[0].Level != LevelSystem {
		t.Errorf("layers[0].Level = %q, want %q", layers[0].Level, LevelSystem)
	}
	if layers[1].Level != LevelUser {
		t.Errorf("layers[1].Level = %q, want %q", layers[1].Level, LevelUser)
	}
}

func TestDiscoverPathsEmptyOverrides(t *testing.T) {
	layers := DiscoverPaths(DiscoverOptions{
		ProjectPath: "./agent-sync.yaml",
		// SystemConfigPath and UserConfigPath empty = use OS defaults.
	})

	// Should have 3 layers: system default, user default, project.
	if len(layers) < 2 {
		t.Fatalf("expected at least 2 layers, got %d", len(layers))
	}
	if layers[len(layers)-1].Level != LevelProject {
		t.Errorf("last layer should be project, got %q", layers[len(layers)-1].Level)
	}
}

func TestDefaultSystemConfigPath(t *testing.T) {
	p := defaultSystemConfigPath()
	if p == "" {
		t.Fatal("system config path should not be empty")
	}

	switch runtime.GOOS {
	case "linux", "darwin":
		if p != "/etc/agent-sync/agent-sync.yaml" {
			t.Errorf("system path = %q, want /etc/agent-sync/agent-sync.yaml", p)
		}
	case "windows":
		if !filepath.IsAbs(p) {
			t.Errorf("system path should be absolute on Windows, got %q", p)
		}
	}
}

func TestDefaultUserConfigPath(t *testing.T) {
	p := defaultUserConfigPath()
	// User config dir may not be available in all test environments.
	if p == "" {
		t.Skip("os.UserConfigDir() not available")
	}
	if !filepath.IsAbs(p) {
		t.Errorf("user path should be absolute, got %q", p)
	}
}

func TestEnvBoolTrue(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{" true ", true},
		{"0", false},
		{"false", false},
		{"", false},
		{"yes", false},
	}

	for _, tt := range tests {
		t.Setenv("TEST_BOOL", tt.value)
		got := envBoolTrue("TEST_BOOL")
		if got != tt.want {
			t.Errorf("envBoolTrue(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

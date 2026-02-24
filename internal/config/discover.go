package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const configFileName = "agent-sync.yaml"
const configDirName = "agent-sync"

// ConfigLevel represents the precedence level of a configuration file.
type ConfigLevel string

const (
	LevelSystem  ConfigLevel = "system"
	LevelUser    ConfigLevel = "user"
	LevelProject ConfigLevel = "project"
)

// ConfigLayerInfo describes a discovered config file and its load status.
type ConfigLayerInfo struct {
	Err    error // non-nil if the file exists but failed to load
	Path   string
	Level  ConfigLevel
	Loaded bool
}

// DiscoverOptions controls how config paths are discovered.
type DiscoverOptions struct {
	// ProjectPath is the project-level config path (required).
	ProjectPath string

	// SystemConfigPath overrides the default system config path.
	// Empty means use the OS default. Set to a nonexistent path to skip.
	SystemConfigPath string

	// UserConfigPath overrides the default user config path.
	// Empty means use the OS default. Set to a nonexistent path to skip.
	UserConfigPath string
}

// DiscoverPaths returns the ordered list of config file paths to check,
// from lowest precedence (system) to highest (project).
// Paths are deduplicated by resolved absolute path.
func DiscoverPaths(opts DiscoverOptions) []ConfigLayerInfo {
	var layers []ConfigLayerInfo
	seen := make(map[string]bool)

	addLayer := func(level ConfigLevel, path string) {
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			abs = path
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		layers = append(layers, ConfigLayerInfo{
			Path:  path,
			Level: level,
		})
	}

	// System-level config.
	sysPath := opts.SystemConfigPath
	if sysPath == "" {
		sysPath = defaultSystemConfigPath()
	}
	addLayer(LevelSystem, sysPath)

	// User-level config.
	userPath := opts.UserConfigPath
	if userPath == "" {
		userPath = defaultUserConfigPath()
	}
	addLayer(LevelUser, userPath)

	// Project-level config (always last, highest precedence).
	addLayer(LevelProject, opts.ProjectPath)

	return layers
}

// defaultSystemConfigPath returns the platform-standard system config path.
func defaultSystemConfigPath() string {
	switch runtime.GOOS {
	case "windows":
		pd := os.Getenv("ProgramData")
		if pd == "" {
			pd = `C:\ProgramData`
		}
		return filepath.Join(pd, configDirName, configFileName)
	default: // linux, darwin, etc.
		return filepath.Join("/etc", configDirName, configFileName)
	}
}

// defaultUserConfigPath returns the platform-standard user config path.
func defaultUserConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, configDirName, configFileName)
}

// EnvNoInherit returns true if AGENT_SYNC_NO_INHERIT is set to "1" or "true".
func EnvNoInherit() bool {
	return envBoolTrue("AGENT_SYNC_NO_INHERIT")
}

// envBoolTrue returns true if the env var is set to "1" or "true" (case-insensitive).
func envBoolTrue(key string) bool {
	v := os.Getenv(key)
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "1" || v == "true"
}

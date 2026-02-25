package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bianoble/agent-sync/internal/cache"
	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/source"
	"github.com/bianoble/agent-sync/internal/target"
)

// loadConfig reads and validates the config file using hierarchical resolution.
// System and user configs are merged below the project config unless --no-inherit
// is set or AGENT_SYNC_NO_INHERIT is enabled.
func loadConfig() (*config.Config, error) {
	result, err := loadConfigHierarchical()
	if err != nil {
		return nil, err
	}
	return result.Config, nil
}

// loadConfigHierarchical returns both the merged config and layer metadata.
func loadConfigHierarchical() (*config.HierarchicalResult, error) {
	inherit := !noInherit && !config.EnvNoInherit()

	opts := config.HierarchicalOptions{
		ProjectPath:      configPath,
		SystemConfigPath: os.Getenv("AGENT_SYNC_SYSTEM_CONFIG"),
		UserConfigPath:   os.Getenv("AGENT_SYNC_USER_CONFIG"),
		NoInherit:        !inherit,
	}

	result, err := config.LoadHierarchical(opts)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if verbose && inherit {
		for _, l := range result.Layers {
			if l.Loaded {
				detail("config: loaded %s %s", l.Level, l.Path)
			} else if l.Err == nil {
				detail("config: %s %s (not found, skipping)", l.Level, l.Path)
			}
		}
		loaded := 0
		for _, l := range result.Layers {
			if l.Loaded {
				loaded++
			}
		}
		if loaded > 1 {
			detail("config: merged %d layers", loaded)
		}
	}

	return result, nil
}

// loadLockfile reads the lockfile if it exists. Returns an empty lockfile if missing.
func loadLockfile() (*lock.Lockfile, error) {
	lf, err := lock.Load(lockfilePath)
	if errors.Is(err, os.ErrNotExist) {
		return &lock.Lockfile{Version: 1}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("loading lockfile %s: %w", lockfilePath, err)
	}
	return lf, nil
}

// saveLockfile writes the lockfile atomically.
func saveLockfile(lf *lock.Lockfile) error {
	return lock.Save(lockfilePath, lf)
}

// projectRoot returns the directory containing the config file.
func projectRoot() (string, error) {
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return "", fmt.Errorf("resolving config path: %w", err)
	}
	return filepath.Dir(abs), nil
}

// newToolMap creates a ToolMap from the config's custom definitions.
func newToolMap(cfg *config.Config) *target.ToolMap {
	return target.NewToolMap(cfg.ToolDefinitions)
}

// newRegistry creates a source registry with all built-in resolvers.
func newRegistry() *source.Registry {
	reg := source.NewRegistry()
	reg.Register("git", &source.GitResolver{})
	reg.Register("url", &source.URLResolver{})
	reg.Register("local", &source.LocalResolver{})
	return reg
}

// newCache creates or opens the content-addressed cache.
func newCache() (*cache.Cache, error) {
	return cache.New(cache.DefaultDir())
}

// info prints a line unless quiet mode is active.
func info(format string, args ...any) {
	if !quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// detail prints a line only in verbose mode.
func detail(format string, args ...any) {
	if verbose {
		fmt.Printf("  "+format+"\n", args...)
	}
}

// errorf prints an error message to stderr.
func errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}

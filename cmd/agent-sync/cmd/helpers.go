package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bianoble/agent-sync/internal/cache"
	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/source"
	"github.com/bianoble/agent-sync/internal/target"
)

// loadConfig reads and validates the config file.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config %s: %w", configPath, err)
	}
	return cfg, nil
}

// loadLockfile reads the lockfile if it exists. Returns an empty lockfile if missing.
func loadLockfile() (*lock.Lockfile, error) {
	lf, err := lock.Load(lockfilePath)
	if os.IsNotExist(err) {
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

// printQuiet returns true if only errors should be shown.
func printQuiet() bool {
	return quiet
}

// printVerbose returns true if detailed output is requested.
func printVerbose() bool {
	return verbose
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

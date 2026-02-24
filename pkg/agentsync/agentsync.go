// Package agentsync provides the public Go library API for agent-sync.
//
// agent-sync is a deterministic, registry-agnostic synchronization system
// for agent files. This package exposes interfaces and constructors for
// embedding agent-sync in other Go programs.
//
// See spec Section 10 for the full library specification.
//
// # Basic Usage
//
//	client, err := agentsync.New(agentsync.Options{
//	    ProjectRoot: "/path/to/project",
//	    ConfigPath:  "agent-sync.yaml",
//	    LockfilePath: "agent-sync.lock",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Update lockfile from upstream
//	result, err := client.Update(ctx, agentsync.UpdateOptions{})
//
//	// Sync files to targets
//	syncResult, err := client.Sync(ctx, agentsync.SyncOptions{})
//
//	// Check for drift
//	checkResult, err := client.Check(ctx)
package agentsync

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bianoble/agent-sync/internal/cache"
	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/engine"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/source"
	"github.com/bianoble/agent-sync/internal/target"
)

// SyncOptions configures a sync operation.
type SyncOptions struct {
	DryRun bool
}

// PruneOptions configures a prune operation.
type PruneOptions struct {
	DryRun bool
}

// Syncer synchronizes files to targets using the lockfile as the source of truth.
// See spec Section 10.1.
type Syncer interface {
	Sync(ctx context.Context, opts SyncOptions) (*SyncResult, error)
}

// Checker verifies that target files match the lockfile.
// See spec Section 10.1.
type Checker interface {
	Check(ctx context.Context) (*CheckResult, error)
}

// Verifier checks whether upstream sources have changed since the lockfile was written.
// See spec Section 10.1.
type Verifier interface {
	Verify(ctx context.Context, sourceNames []string) (*VerifyResult, error)
}

// Pruner removes files no longer referenced in the configuration.
// See spec Section 10.1.
type Pruner interface {
	Prune(ctx context.Context, opts PruneOptions) (*PruneResult, error)
}

// UpdateOptions configures an update operation.
type UpdateOptions struct {
	SourceNames []string // empty = update all
	DryRun      bool
}

// UpdateResult holds the outcome of an update operation.
type UpdateResult struct {
	Updated []SourceUpdate
	Failed  []SourceError
}

// SourceUpdate records what changed for a single source during update.
type SourceUpdate struct {
	Name   string
	Before string // human-readable summary of previous state
	After  string // human-readable summary of new state
}

// Updater resolves sources against upstream and updates the lockfile.
type Updater interface {
	Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error)
}

// Options configures an agent-sync client.
type Options struct {
	// ProjectRoot is the directory containing agent-sync.yaml.
	// If empty, defaults to the directory containing ConfigPath.
	ProjectRoot string

	// ConfigPath is the path to the config file. Default: "agent-sync.yaml".
	ConfigPath string

	// LockfilePath is the path to the lockfile. Default: "agent-sync.lock".
	LockfilePath string

	// CacheDir is the cache directory. If empty, uses the default (~/.cache/agent-sync).
	CacheDir string
}

// Client is the main entry point for the agent-sync library.
// It implements Syncer, Checker, Verifier, Pruner, and Updater.
type Client struct {
	registry     *source.Registry
	cache        *cache.Cache
	projectRoot  string
	configPath   string
	lockfilePath string
}

// New creates a new agent-sync Client.
func New(opts Options) (*Client, error) {
	if opts.ConfigPath == "" {
		opts.ConfigPath = "agent-sync.yaml"
	}
	if opts.LockfilePath == "" {
		opts.LockfilePath = "agent-sync.lock"
	}

	root := opts.ProjectRoot
	if root == "" {
		abs, err := filepath.Abs(opts.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("resolving config path: %w", err)
		}
		root = filepath.Dir(abs)
	}

	cacheDir := opts.CacheDir
	if cacheDir == "" {
		cacheDir = cache.DefaultDir()
	}
	c, err := cache.New(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("initializing cache: %w", err)
	}

	reg := source.NewRegistry()
	reg.Register("git", &source.GitResolver{})
	reg.Register("url", &source.URLResolver{})
	reg.Register("local", &source.LocalResolver{})

	return &Client{
		projectRoot:  root,
		configPath:   opts.ConfigPath,
		lockfilePath: opts.LockfilePath,
		registry:     reg,
		cache:        c,
	}, nil
}

func (c *Client) loadConfig() (*config.Config, error) {
	return config.Load(c.configPath)
}

func (c *Client) loadLockfile() (*lock.Lockfile, error) {
	lf, err := lock.Load(c.lockfilePath)
	if err != nil {
		return &lock.Lockfile{Version: 1}, nil //nolint: nilerr
	}
	return lf, nil
}

func (c *Client) toolMap(cfg *config.Config) *target.ToolMap {
	return target.NewToolMap(cfg.ToolDefinitions)
}

// Sync synchronizes files to targets using the lockfile as the source of truth.
func (c *Client) Sync(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	cfg, err := c.loadConfig()
	if err != nil {
		return nil, err
	}
	lf, err := c.loadLockfile()
	if err != nil {
		return nil, err
	}

	eng := &engine.SyncEngine{
		Registry:    c.registry,
		Cache:       c.cache,
		ToolMap:     c.toolMap(cfg),
		ProjectRoot: c.projectRoot,
	}

	return eng.Sync(ctx, *lf, *cfg, engine.SyncOptions{DryRun: opts.DryRun})
}

// Check verifies that target files match the lockfile.
func (c *Client) Check(ctx context.Context) (*CheckResult, error) {
	cfg, err := c.loadConfig()
	if err != nil {
		return nil, err
	}
	lf, err := c.loadLockfile()
	if err != nil {
		return nil, err
	}

	eng := &engine.CheckEngine{
		ToolMap:     c.toolMap(cfg),
		ProjectRoot: c.projectRoot,
	}

	return eng.Check(ctx, *lf, *cfg)
}

// Verify checks whether upstream sources have changed since the lockfile was written.
func (c *Client) Verify(ctx context.Context, sourceNames []string) (*VerifyResult, error) {
	cfg, err := c.loadConfig()
	if err != nil {
		return nil, err
	}
	lf, err := c.loadLockfile()
	if err != nil {
		return nil, err
	}

	eng := &engine.VerifyEngine{
		Registry:    c.registry,
		ProjectRoot: c.projectRoot,
	}

	return eng.Verify(ctx, *lf, *cfg, sourceNames)
}

// Prune removes files no longer referenced in the configuration.
func (c *Client) Prune(ctx context.Context, opts PruneOptions) (*PruneResult, error) {
	cfg, err := c.loadConfig()
	if err != nil {
		return nil, err
	}
	lf, err := c.loadLockfile()
	if err != nil {
		return nil, err
	}

	eng := &engine.PruneEngine{
		ToolMap:     c.toolMap(cfg),
		ProjectRoot: c.projectRoot,
	}

	return eng.Prune(ctx, *lf, *cfg, engine.PruneOptions{DryRun: opts.DryRun})
}

// Update resolves sources against upstream and updates the lockfile.
func (c *Client) Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
	cfg, err := c.loadConfig()
	if err != nil {
		return nil, err
	}
	lf, err := c.loadLockfile()
	if err != nil {
		return nil, err
	}

	eng := &engine.UpdateEngine{
		Registry:    c.registry,
		Cache:       c.cache,
		ProjectRoot: c.projectRoot,
	}

	engineOpts := engine.UpdateOptions{
		DryRun:      opts.DryRun,
		AutoConfirm: true, // library always auto-confirms
		SourceNames: opts.SourceNames,
	}

	result, err := eng.Update(ctx, *cfg, lf, engineOpts)
	if err != nil {
		return nil, err
	}

	// Convert engine result to public API result.
	out := &UpdateResult{}
	for _, u := range result.Updated {
		su := SourceUpdate{Name: u.Name}
		if u.Before != nil {
			su.Before = summarizeLocked(*u.Before)
		} else {
			su.Before = "(new)"
		}
		if u.After != nil {
			su.After = summarizeLocked(*u.After)
		}
		out.Updated = append(out.Updated, su)
	}
	out.Failed = append(out.Failed, result.Failed...)

	// Save lockfile if not dry-run.
	if !opts.DryRun && result.Lockfile != nil {
		if err := lock.Save(c.lockfilePath, result.Lockfile); err != nil {
			return nil, fmt.Errorf("saving lockfile: %w", err)
		}
	}

	return out, nil
}

func summarizeLocked(ls lock.LockedSource) string {
	if ls.Resolved.Commit != "" {
		short := ls.Resolved.Commit
		if len(short) > 8 {
			short = short[:8]
		}
		return short
	}
	if ls.Resolved.SHA256 != "" {
		short := ls.Resolved.SHA256
		if len(short) > 8 {
			short = short[:8]
		}
		return "sha256:" + short
	}
	if len(ls.Resolved.Files) > 0 {
		return fmt.Sprintf("(%d files)", len(ls.Resolved.Files))
	}
	return "(unknown)"
}

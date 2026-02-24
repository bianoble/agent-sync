package engine

import (
	"context"
	"fmt"

	"github.com/bianoble/agent-sync/internal/cache"
	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/source"
	"github.com/bianoble/agent-sync/pkg/agentsync"
)

// UpdateEngine resolves sources against their upstream and updates the lockfile.
type UpdateEngine struct {
	Registry    *source.Registry
	Cache       *cache.Cache
	ProjectRoot string
}

// UpdateOptions configures an update operation.
type UpdateOptions struct {
	DryRun      bool
	AutoConfirm bool
	SourceNames []string // empty = update all
}

// SourceUpdate records what changed for a single source.
type SourceUpdate struct {
	Name   string
	Before *lock.LockedSource
	After  *lock.LockedSource
}

// UpdateResult holds the outcome of an update operation.
type UpdateResult struct {
	Updated  []SourceUpdate
	Failed   []agentsync.SourceError
	Lockfile *lock.Lockfile // nil if dry-run
}

// Update resolves sources and updates the lockfile.
func (e *UpdateEngine) Update(ctx context.Context, cfg config.Config, currentLock *lock.Lockfile, opts UpdateOptions) (*UpdateResult, error) {
	result := &UpdateResult{}

	// Determine which sources to update.
	sourcesToUpdate := cfg.Sources
	if len(opts.SourceNames) > 0 {
		filtered := make([]config.Source, 0, len(opts.SourceNames))
		configByName := make(map[string]config.Source)
		for _, s := range cfg.Sources {
			configByName[s.Name] = s
		}
		for _, name := range opts.SourceNames {
			s, ok := configByName[name]
			if !ok {
				result.Failed = append(result.Failed, agentsync.SourceError{
					Source: name,
					Err:    fmt.Errorf("source '%s' not found in config", name),
				})
				continue
			}
			filtered = append(filtered, s)
		}
		sourcesToUpdate = filtered
	}

	// Build lookup of current locked sources.
	currentByName := make(map[string]lock.LockedSource)
	if currentLock != nil {
		for _, ls := range currentLock.Sources {
			currentByName[ls.Name] = ls
		}
	}

	// Resolve each source.
	newByName := make(map[string]lock.LockedSource)
	for _, src := range sourcesToUpdate {
		resolver, err := e.Registry.Get(src.Type)
		if err != nil {
			result.Failed = append(result.Failed, agentsync.SourceError{Source: src.Name, Err: err})
			continue
		}

		resolved, err := resolver.Resolve(ctx, src, e.ProjectRoot)
		if err != nil {
			result.Failed = append(result.Failed, agentsync.SourceError{Source: src.Name, Err: err})
			continue
		}

		// Convert to lockfile entry.
		ls := resolvedToLocked(src, resolved)

		// Record update.
		var before *lock.LockedSource
		if prev, ok := currentByName[src.Name]; ok {
			before = &prev
		}
		result.Updated = append(result.Updated, SourceUpdate{
			Name:   src.Name,
			Before: before,
			After:  &ls,
		})

		newByName[src.Name] = ls

		// Cache fetched content.
		if e.Cache != nil {
			fetched, fetchErr := resolver.Fetch(ctx, resolved)
			if fetchErr == nil {
				for _, f := range fetched {
					_ = e.Cache.Put(f.SHA256, f.Content)
				}
			}
		}
	}

	if opts.DryRun {
		return result, nil
	}

	// Build new lockfile: updated sources get new state, failed keep old, others unchanged.
	newLock := &lock.Lockfile{Version: 1}

	// Track which sources we've handled.
	handled := make(map[string]bool)

	// Add all config sources in order.
	for _, src := range cfg.Sources {
		handled[src.Name] = true
		if ls, ok := newByName[src.Name]; ok {
			newLock.Sources = append(newLock.Sources, ls)
		} else if ls, ok := currentByName[src.Name]; ok {
			newLock.Sources = append(newLock.Sources, ls)
		}
		// else: new source that failed resolution — skip
	}

	result.Lockfile = newLock
	return result, nil
}

func resolvedToLocked(src config.Source, resolved *source.ResolvedSource) lock.LockedSource {
	ls := lock.LockedSource{
		Name:   src.Name,
		Type:   src.Type,
		Repo:   src.Repo,
		Status: "ok",
	}

	ls.Resolved.Commit = resolved.Commit
	ls.Resolved.Tree = resolved.Tree
	ls.Resolved.URL = resolved.URL
	ls.Resolved.Path = resolved.Path
	ls.Resolved.SHA256 = resolvedSHA256(resolved)

	if len(resolved.Files) > 0 {
		ls.Resolved.Files = make(map[string]lock.FileHash)
		for path, hash := range resolved.Files {
			ls.Resolved.Files[path] = lock.FileHash{SHA256: hash}
		}
	}

	return ls
}

func resolvedSHA256(resolved *source.ResolvedSource) string {
	// For URL sources, the SHA256 is the single file hash.
	if resolved.Type == "url" {
		for _, hash := range resolved.Files {
			return hash
		}
	}
	return ""
}

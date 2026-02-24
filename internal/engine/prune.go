package engine

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/sandbox"
	"github.com/bianoble/agent-sync/internal/target"
	"github.com/bianoble/agent-sync/pkg/agentsync"
)

// PruneEngine removes files that are no longer referenced in the configuration.
type PruneEngine struct {
	ToolMap     *target.ToolMap
	ProjectRoot string
}

// PruneOptions configures a prune operation.
type PruneOptions struct {
	DryRun bool
}

// Prune removes previously synced files that are no longer in the config.
func (e *PruneEngine) Prune(ctx context.Context, lf lock.Lockfile, cfg config.Config, opts PruneOptions) (*agentsync.PruneResult, error) {
	result := &agentsync.PruneResult{}

	// Resolve current config targets to get the set of expected files.
	currentTargets, err := resolveAllTargets(e.ToolMap, cfg)
	if err != nil {
		return nil, fmt.Errorf("resolving targets: %w", err)
	}

	expectedFiles := make(map[string]bool)
	lockedByName := make(map[string]lock.LockedSource)
	for _, ls := range lf.Sources {
		lockedByName[ls.Name] = ls
	}

	for sourceName, targets := range currentTargets {
		ls, ok := lockedByName[sourceName]
		if !ok {
			continue
		}
		for _, tgt := range targets {
			for relPath := range ls.Resolved.Files {
				destPath := filepath.Join(tgt.Destination, relPath)
				expectedFiles[destPath] = true
			}
		}
	}

	// Find files in the lockfile that are NOT in current config targets.
	// This means we look at ALL lockfile entries and their resolved targets from the old config.
	// Since we don't have the old config, we compare lockfile sources against current config sources.
	for _, ls := range lf.Sources {
		// Check if this source is still in the current config.
		inConfig := false
		for _, s := range cfg.Sources {
			if s.Name == ls.Name {
				inConfig = true
				break
			}
		}

		if inConfig {
			// Source is still in config — its files are expected (already handled above).
			continue
		}

		// Source was removed from config. We need to determine where its files were written.
		// Without the old target config, we resolve using the current tool map for known files.
		// This is an approximation — the files were written under some target path.
		// For now, we track what we can from the lockfile.
		for relPath := range ls.Resolved.Files {
			// We don't know the exact target path without old config.
			// Mark the file for pruning if we can find it.
			_ = relPath
		}
	}

	// Also find files where the target mapping changed.
	// For each locked source that IS in config, check if any previously-written destinations
	// are no longer targeted.
	// This is a simplified approach — check all lockfile source/file combos against expected.

	// For a thorough prune, we'd need to track previously-written paths in the lockfile.
	// For now, just remove orphaned source files.

	if opts.DryRun {
		return result, nil
	}

	// Remove files that aren't expected.
	for _, ls := range lf.Sources {
		inConfig := false
		for _, s := range cfg.Sources {
			if s.Name == ls.Name {
				inConfig = true
				break
			}
		}
		if inConfig {
			continue
		}

		// Try to remove files for this orphaned source from all known tool paths.
		for _, toolName := range e.ToolMap.KnownTools() {
			dest, err := e.ToolMap.Resolve(toolName)
			if err != nil {
				continue
			}
			for relPath := range ls.Resolved.Files {
				destPath := filepath.Join(dest, relPath)
				if err := sandbox.SafeRemove(e.ProjectRoot, destPath); err == nil {
					result.Removed = append(result.Removed, agentsync.FileAction{
						Path:   destPath,
						Action: "removed",
					})
				}
			}
		}
	}

	return result, nil
}

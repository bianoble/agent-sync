package engine

import (
	"context"
	"fmt"

	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/source"
	"github.com/bianoble/agent-sync/pkg/agentsync"
)

// VerifyEngine checks whether upstream sources have changed since the lockfile was written.
type VerifyEngine struct {
	Registry    *source.Registry
	ProjectRoot string
}

// Verify checks upstream sources against lockfile state.
func (e *VerifyEngine) Verify(ctx context.Context, lf lock.Lockfile, cfg config.Config, sourceNames []string) (*agentsync.VerifyResult, error) {
	result := &agentsync.VerifyResult{}

	// Build lookup.
	lockedByName := make(map[string]lock.LockedSource)
	for _, ls := range lf.Sources {
		lockedByName[ls.Name] = ls
	}

	configByName := make(map[string]config.Source)
	for _, s := range cfg.Sources {
		configByName[s.Name] = s
	}

	// Determine which sources to verify.
	names := sourceNames
	if len(names) == 0 {
		for _, s := range cfg.Sources {
			names = append(names, s.Name)
		}
	}

	for _, name := range names {
		src, ok := configByName[name]
		if !ok {
			result.Errors = append(result.Errors, agentsync.SourceError{
				Source: name,
				Err:    fmt.Errorf("source '%s' not found in config", name),
			})
			continue
		}

		ls, ok := lockedByName[name]
		if !ok {
			result.Changed = append(result.Changed, agentsync.SourceDelta{
				Source: name,
				Before: "(not locked)",
				After:  "(needs update)",
			})
			continue
		}

		resolver, err := e.Registry.Get(src.Type)
		if err != nil {
			result.Errors = append(result.Errors, agentsync.SourceError{Source: name, Err: err})
			continue
		}

		resolved, err := resolver.Resolve(ctx, src, e.ProjectRoot)
		if err != nil {
			result.Errors = append(result.Errors, agentsync.SourceError{Source: name, Err: err})
			continue
		}

		if hasChanged(ls, resolved) {
			result.Changed = append(result.Changed, agentsync.SourceDelta{
				Source: name,
				Before: summarizeLocked(ls),
				After:  summarizeResolved(resolved),
			})
		} else {
			result.UpToDate = append(result.UpToDate, name)
		}
	}

	return result, nil
}

func hasChanged(ls lock.LockedSource, resolved *source.ResolvedSource) bool {
	// For git: compare commit SHA.
	if ls.Type == "git" && resolved.Commit != "" && ls.Resolved.Commit != resolved.Commit {
		return true
	}

	// For url/local: compare file hashes.
	if len(ls.Resolved.Files) != len(resolved.Files) {
		return true
	}
	for path, fh := range ls.Resolved.Files {
		if newHash, ok := resolved.Files[path]; !ok || fh.SHA256 != newHash {
			return true
		}
	}

	return false
}

func summarizeLocked(ls lock.LockedSource) string {
	switch ls.Type {
	case "git":
		if ls.Resolved.Commit != "" {
			short := ls.Resolved.Commit
			if len(short) > 8 {
				short = short[:8]
			}
			return short
		}
	case "url":
		if ls.Resolved.SHA256 != "" {
			short := ls.Resolved.SHA256
			if len(short) > 8 {
				short = short[:8]
			}
			return "sha256:" + short
		}
	case "local":
		return fmt.Sprintf("(%d files)", len(ls.Resolved.Files))
	}
	return "(unknown)"
}

func summarizeResolved(resolved *source.ResolvedSource) string {
	switch resolved.Type {
	case "git":
		if resolved.Commit != "" {
			short := resolved.Commit
			if len(short) > 8 {
				short = short[:8]
			}
			return short
		}
	case "url":
		for _, hash := range resolved.Files {
			short := hash
			if len(short) > 8 {
				short = short[:8]
			}
			return "sha256:" + short
		}
	case "local":
		return fmt.Sprintf("(%d files)", len(resolved.Files))
	}
	return "(unknown)"
}

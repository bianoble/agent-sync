package engine

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/target"
)

// StatusEngine computes the state of all synced sources.
type StatusEngine struct {
	ToolMap     *target.ToolMap
	ProjectRoot string
}

// SourceStatus describes the current state of a source.
type SourceStatus struct {
	Name     string
	Type     string
	PinnedAt string
	Targets  []string
	State    string // "synced", "drifted", "missing", "pending"
}

// Status returns the state of all (or named) sources.
func (e *StatusEngine) Status(ctx context.Context, lf lock.Lockfile, cfg config.Config, sourceNames []string) ([]SourceStatus, error) {
	targetMap, err := resolveAllTargets(e.ToolMap, cfg)
	if err != nil {
		return nil, err
	}

	lockedByName := make(map[string]lock.LockedSource)
	for _, ls := range lf.Sources {
		lockedByName[ls.Name] = ls
	}

	names := sourceNames
	if len(names) == 0 {
		for _, s := range cfg.Sources {
			names = append(names, s.Name)
		}
	}

	var statuses []SourceStatus
	for _, name := range names {
		ls, locked := lockedByName[name]
		targets := targetMap[name]

		s := SourceStatus{
			Name: name,
			Type: sourceTypeFromConfig(name, cfg),
		}

		// Gather target paths.
		for _, tgt := range targets {
			s.Targets = append(s.Targets, tgt.Destination)
		}

		if !locked {
			s.State = "pending"
			s.PinnedAt = "(not locked)"
		} else {
			s.PinnedAt = summarizeLocked(ls)
			s.State = computeState(e.ProjectRoot, ls, targets)
		}

		statuses = append(statuses, s)
	}

	return statuses, nil
}

func sourceTypeFromConfig(name string, cfg config.Config) string {
	for _, s := range cfg.Sources {
		if s.Name == name {
			return s.Type
		}
	}
	return ""
}

func computeState(projectRoot string, ls lock.LockedSource, targets []target.ResolvedTarget) string {
	anyMissing := false
	anyDrifted := false

	for _, tgt := range targets {
		for relPath, fh := range ls.Resolved.Files {
			destPath := filepath.Join(tgt.Destination, relPath)
			absPath := filepath.Join(projectRoot, destPath)

			content, err := os.ReadFile(absPath)
			if err != nil {
				anyMissing = true
				continue
			}

			actualHash := sha256Hex(content)
			if actualHash != fh.SHA256 {
				anyDrifted = true
			}
		}
	}

	if anyMissing {
		return "missing"
	}
	if anyDrifted {
		return "drifted"
	}
	return "synced"
}

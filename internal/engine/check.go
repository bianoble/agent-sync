package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/bianoble/agent-sync/internal/target"
	"github.com/bianoble/agent-sync/pkg/agentsync"
)

// CheckEngine verifies that target files match the lockfile.
type CheckEngine struct {
	ToolMap     *target.ToolMap
	ProjectRoot string
}

// Check verifies target files against the lockfile.
// Returns Clean=true if everything matches.
func (e *CheckEngine) Check(ctx context.Context, lf lock.Lockfile, cfg config.Config) (*agentsync.CheckResult, error) {
	result := &agentsync.CheckResult{Clean: true}

	// Resolve all targets.
	targetMap, err := resolveAllTargets(e.ToolMap, cfg)
	if err != nil {
		return nil, err
	}

	// Build locked source lookup.
	lockedByName := make(map[string]lock.LockedSource)
	for _, ls := range lf.Sources {
		lockedByName[ls.Name] = ls
	}

	// For each source with targets, check that files exist and match.
	for sourceName, targets := range targetMap {
		ls, ok := lockedByName[sourceName]
		if !ok {
			continue // no lockfile entry, skip
		}

		for _, tgt := range targets {
			for relPath, fh := range ls.Resolved.Files {
				destPath := filepath.Join(tgt.Destination, relPath)
				absPath := filepath.Join(e.ProjectRoot, destPath)

				content, readErr := os.ReadFile(absPath)
				if readErr != nil {
					if os.IsNotExist(readErr) {
						result.Missing = append(result.Missing, destPath)
						result.Clean = false
					}
					continue
				}

				actualHash := sha256Hex(content)
				if actualHash != fh.SHA256 {
					result.Drifted = append(result.Drifted, agentsync.DriftEntry{
						Path:     destPath,
						Expected: fh.SHA256,
						Actual:   actualHash,
					})
					result.Clean = false
				}
			}
		}
	}

	return result, nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

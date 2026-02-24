package agentsync

import "github.com/bianoble/agent-sync/internal/engine"

// Type aliases re-export engine result types as the public API.
// Users import "github.com/bianoble/agent-sync/pkg/agentsync" and use
// agentsync.SyncResult, agentsync.CheckResult, etc.

type FileAction = engine.FileAction
type SourceError = engine.SourceError
type DriftEntry = engine.DriftEntry
type SourceDelta = engine.SourceDelta
type SyncResult = engine.SyncResult
type CheckResult = engine.CheckResult
type VerifyResult = engine.VerifyResult
type PruneResult = engine.PruneResult

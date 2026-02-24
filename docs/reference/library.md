# Go Library Reference

agent-sync is usable as a Go library via the `pkg/agentsync` package.

## Installation

```bash
go get github.com/bianoble/agent-sync@latest
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/bianoble/agent-sync/pkg/agentsync"
)

func main() {
    client, err := agentsync.New(agentsync.Options{
        ProjectRoot: ".",
        ConfigPath:  "agent-sync.yaml",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Update lockfile from upstream sources
    updateResult, err := client.Update(ctx, agentsync.UpdateOptions{})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Updated %d sources\n", len(updateResult.Updated))

    // Sync files to targets
    syncResult, err := client.Sync(ctx, agentsync.SyncOptions{})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Written: %d, Skipped: %d\n",
        len(syncResult.Written), len(syncResult.Skipped))

    // Check for drift
    checkResult, err := client.Check(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if checkResult.Clean {
        fmt.Println("All files match lockfile")
    }
}
```

## Client Options

```go
type Options struct {
    ProjectRoot  string // Directory containing agent-sync.yaml
    ConfigPath   string // Default: "agent-sync.yaml"
    LockfilePath string // Default: "agent-sync.lock"
    CacheDir     string // Default: ~/.cache/agent-sync
}
```

## Interfaces

The `Client` type implements all of these interfaces:

### Syncer

```go
type Syncer interface {
    Sync(ctx context.Context, opts SyncOptions) (*SyncResult, error)
}
```

### Checker

```go
type Checker interface {
    Check(ctx context.Context) (*CheckResult, error)
}
```

### Verifier

```go
type Verifier interface {
    Verify(ctx context.Context, sourceNames []string) (*VerifyResult, error)
}
```

### Pruner

```go
type Pruner interface {
    Prune(ctx context.Context, opts PruneOptions) (*PruneResult, error)
}
```

### Updater

```go
type Updater interface {
    Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error)
}
```

## Result Types

### SyncResult

```go
type SyncResult struct {
    Written []FileAction  // Files that were written or modified
    Skipped []FileAction  // Files that were unchanged
    Errors  []SourceError // Per-source errors
}
```

### CheckResult

```go
type CheckResult struct {
    Clean   bool         // True if all files match
    Drifted []DriftEntry // Files that have changed
    Missing []string     // Files that are missing
}
```

### VerifyResult

```go
type VerifyResult struct {
    UpToDate []string      // Sources matching upstream
    Changed  []SourceDelta // Sources with upstream changes
    Errors   []SourceError // Resolution errors
}
```

### PruneResult

```go
type PruneResult struct {
    Removed []FileAction  // Files that were removed
    Errors  []SourceError // Per-source errors
}
```

## Library Rules

- The library does **not** depend on the CLI
- All methods accept `context.Context` for cancellation and timeout
- All methods return structured errors, not exit codes
- Update operations auto-confirm (no interactive prompts)

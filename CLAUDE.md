# CLAUDE.md

This file provides context for AI coding assistants working on the agent-sync codebase.

## Project Overview

agent-sync is a deterministic, registry-agnostic synchronization system for agent files. It fetches files from external sources (Git, URL, local), pins them via a lockfile, and syncs them into tool-specific directories.

## Build & Test

```bash
make build          # Compile to bin/agent-sync with ldflags
make test           # Run tests with -race
make lint           # golangci-lint (see .golangci.yml for config)
make vet            # go vet ./...
make fmt            # gofmt + goimports
```

Or directly:

```bash
go build ./cmd/agent-sync
go test -race ./...
```

## Project Structure

```
cmd/agent-sync/cmd/      CLI commands (Cobra). root.go defines global flags.
internal/
  config/                Config loading and hierarchical discovery/merge.
    discover.go          Three-level config discovery (system/user/project).
    load.go              Load, parse, merge, validate YAML configs.
  cache/                 Content-addressed cache keyed by SHA256.
  engine/                Core operations. Each engine function takes lockfile + config.
    sync.go              Sync files to targets (with rollback on failure).
    update.go            Resolve sources and update lockfile.
    check.go             Verify target files match lockfile hashes.
    verify.go            Check if upstream sources have changed.
    status.go            Report sync state per source.
    prune.go             Remove orphaned files.
  lock/                  Lockfile YAML read/write/validate.
  sandbox/               Safe-write enforcement. Prevents path traversal, symlinks.
  source/                Source resolvers (git, url, local). FS abstraction (OSFS).
  target/                Tool map resolution. Maps tool names → directory paths.
  transform/             Template variable substitution and file overrides.
pkg/agentsync/           Public Go library. Client wraps all engine operations.
```

## Key Architectural Decisions

- **Content-addressed cache**: Files cached by SHA256 hash in `~/.cache/agent-sync/`. Cache hits skip network fetches.
- **Atomic writes**: All file writes use temp file + rename to prevent partial writes.
- **Hierarchical config**: System (`/etc/agent-sync/`) → User (`~/.config/agent-sync/`) → Project (`./agent-sync.yaml`). Merge semantics: sources merge by name, targets concatenate, variables deep-merge.
- **Engine pattern**: Each operation (sync, update, check, etc.) is a standalone function in `internal/engine/` that accepts a lockfile, config, and options. The CLI and library both call through these engines.
- **Resolver interface**: `internal/source/` defines a `Resolver` interface. Git, URL, and Local each implement it. The `HTTPClient` and `FS` interfaces allow test doubles.

## Coding Conventions

- **Error wrapping**: Use `fmt.Errorf("context: %w", err)` consistently.
- **Error checking**: Always use `errors.Is(err, os.ErrNotExist)`, never `os.IsNotExist(err)`. The latter does not unwrap `%w`-wrapped errors.
- **Testing**: Use `t.TempDir()` for temp directories, `t.Setenv()` for env vars, table-driven tests. Skip permission tests when root: `if os.Getuid() == 0 { t.Skip("test unreliable as root") }`.
- **Formatting**: gofmt + goimports. golangci-lint must pass clean.
- **No import cycles**: `pkg/agentsync` imports `internal/` packages. CLI imports `pkg/agentsync`. Internal packages do not import CLI or `pkg/`.

## Config Merge Semantics

| Field | Merge Strategy |
|-------|----------------|
| `version` | Must agree across all layers |
| `variables` | Deep merge (higher-precedence key wins) |
| `sources` | Merge by `name` (project replaces system) |
| `tool_definitions` | Merge by `name` |
| `targets` | Concatenate (system, then user, then project) |
| `overrides` | Concatenate |
| `transforms` | Concatenate |

## Spec Version

The project implements spec v0.4 (see `docs/spec.md`). Config and lockfile both use `version: 1`.

# agent-sync Specification v0.4

> This is the authoritative specification. The file `agent-sync-spec-v0.4.md` was a historical duplicate and has been removed.

**Status:** Draft
**Audience:** Library implementers, CLI authors, and platform integrators
**Scope:** Deterministic synchronization of agent-related files from external sources into a project workspace.
**Non-Goals:** Agent execution, registry hosting, agent runtime orchestration.

---

# 1. Overview

## 1.1 Purpose

`agent-sync` is a **deterministic, registry-agnostic dependency and synchronization system for agent files**.

It enables projects to:

* Fetch agent files from external sources (Git, URL, local)
* Pin those files immutably
* Apply controlled transformations and overlays
* Safely synchronize them into tool-specific or project-specific locations

Core principle:

> agent-sync manages **files as immutable inputs and deterministic outputs**

It does not interpret agent semantics.

---

## 1.2 Design Goals

### Primary Goals

* Deterministic
* Secure by default
* Registry-agnostic
* Tool-agnostic
* Minimal and composable
* Embeddable as a Go library

### Secondary Goals

* Human-readable configuration
* Explicit updates only
* CI-friendly drift detection

---

## 1.3 Non-Goals

agent-sync does NOT:

* Execute agents
* Interpret agent behavior
* Provide a registry (deferred)
* Replace configuration management tools
* Replace package managers

---

# 2. Conceptual Model

agent-sync operates on four layers:

```
Sources → Locked Inputs → Transform → Target Files
```

---

## 2.1 Sources

Sources provide files.

Supported source types:

* git
* url
* local

Each source is resolved into an immutable artifact.

---

## 2.2 Lockfile

The lockfile records the **fully resolved, immutable state** of all inputs.

The lockfile guarantees:

* Reproducible sync
* Supply chain integrity
* Drift detection

---

## 2.3 Transform

Optional transforms include:

* Template variable substitution
* Overlay merging
* File mapping
* Custom transform hooks

Transforms MUST be deterministic.

---

## 2.4 Targets

Targets define where files are written.

Targets MUST remain inside the project root sandbox.

Targets may be specified explicitly or derived from the tool map (see Section 3.2).

---

# 3. Configuration File

File name:

```
agent-sync.yaml
```

---

## 3.1 Example

```yaml
version: 1

variables:
  project: my-project
  language: go

sources:

  - name: base-rules
    type: git
    repo: https://github.com/org/rules.git
    ref: v1.2.0
    paths:
      - core/

  - name: security-policy
    type: url
    url: https://example.com/policy.md
    checksum: sha256:abcdef...

  - name: team-standards
    type: local
    path: ./agents/standards/

targets:

  - source: base-rules
    tools: [cursor, claude-code, copilot]

  - source: security-policy
    tools: [cursor, claude-code]

  - source: team-standards
    destination: .custom/agents/

overrides:

  - target: security.md
    strategy: append
    file: local/security-extension.md

transforms:

  - source: base-rules
    type: template
    vars:
      project: "{{ .project }}"

  - source: team-standards
    type: custom
    command: "./scripts/merge-yaml.sh"
    output_hash: sha256:expected...
```

---

## 3.2 Tool Map

The tool map provides automatic path resolution for known agent tools.

### Built-in Tool Definitions

agent-sync ships with built-in path mappings for known tools:

| Tool          | Default Destination        |
|---------------|----------------------------|
| cursor        | .cursor/rules/             |
| claude-code   | .claude/                   |
| copilot       | .github/copilot/           |
| windsurf      | .windsurf/rules/           |
| cline         | .cline/rules/              |
| codex         | .codex/                    |

### Custom Tool Definitions

Teams may define custom tools or override built-in paths:

```yaml
tool_definitions:

  - name: internal-agent
    destination: .internal/agent-config/

  - name: cursor
    destination: .cursor/custom-rules/
```

### Tool Map Resolution Rules

1. If a target specifies `tools:`, agent-sync resolves each tool name to its destination path.
2. Custom `tool_definitions` override built-in mappings.
3. If a tool name has no built-in or custom definition, agent-sync MUST error with a clear message indicating the unknown tool and how to define it.
4. If a target specifies `destination:` directly, no tool map resolution occurs — the path is used as-is.
5. `tools:` and `destination:` are mutually exclusive on a single target entry.

---

## 3.3 Hierarchical Configuration

agent-sync supports three-level hierarchical configuration: **system**, **user**, and **project**. Configs are discovered automatically and merged in order of increasing precedence.

### Discovery Paths

| Level | macOS | Linux | Windows |
|-------|-------|-------|---------|
| System | `/etc/agent-sync/agent-sync.yaml` | `/etc/agent-sync/agent-sync.yaml` | `%ProgramData%\agent-sync\agent-sync.yaml` |
| User | `~/Library/Application Support/agent-sync/agent-sync.yaml` | `$XDG_CONFIG_HOME/agent-sync/agent-sync.yaml` | `%AppData%\agent-sync\agent-sync.yaml` |
| Project | `./agent-sync.yaml` | `./agent-sync.yaml` | `.\agent-sync.yaml` |

Missing config files at any level are silently skipped.

### Merge Semantics

When multiple config levels are present, they are merged as follows:

| Field | Strategy |
|-------|----------|
| `version` | MUST agree across all layers. Mismatch is a fatal error. |
| `variables` | Deep merge. Higher-precedence keys overwrite lower. Unique keys are preserved. |
| `sources` | Merge by `name`. A source in a higher-precedence layer fully replaces a source with the same name from a lower layer. |
| `tool_definitions` | Merge by `name`. Same replacement semantics as sources. |
| `targets` | Concatenate. System targets first, then user, then project. |
| `overrides` | Concatenate. Applied in order: system, user, project. |
| `transforms` | Concatenate. Applied in order: system, user, project. |

### Disabling Hierarchical Resolution

The `--no-inherit` CLI flag or `AGENT_SYNC_NO_INHERIT=1` environment variable disables hierarchical resolution. When set, only the project-level config is used. This is RECOMMENDED for CI/CD environments to ensure reproducible builds.

### Environment Variable Overrides

| Variable | Purpose |
|----------|---------|
| `AGENT_SYNC_SYSTEM_CONFIG` | Override the system config file path |
| `AGENT_SYNC_USER_CONFIG` | Override the user config file path |
| `AGENT_SYNC_NO_INHERIT` | Set to `1` or `true` to disable hierarchical resolution |

---

# 4. Lockfile Specification

File:

```
agent-sync.lock
```

---

## 4.1 Purpose

The lockfile records:

* Fully resolved source identity
* Content hashes
* Transform output hashes
* Sync status per source

The lockfile is the authoritative record.

---

## 4.2 Example

```yaml
version: 1

sources:

  - name: base-rules
    type: git
    repo: https://github.com/org/rules.git

    resolved:

      commit: 3f8c9abf...
      tree: a8bcdef...

      files:

        core/security.md:
          sha256: 123abc...

        core/general.md:
          sha256: 456def...

    status: ok

  - name: security-policy
    type: url

    resolved:

      url: https://example.com/policy.md
      sha256: abcdef123...

    status: ok

  - name: team-standards
    type: local

    resolved:

      path: ./agents/standards/

      files:

        standards/naming.md:
          sha256: 789ghi...

        standards/testing.md:
          sha256: 012jkl...

    status: ok
```

---

## 4.3 Lockfile Guarantees

Given the same lockfile, agent-sync MUST produce:

* Same inputs
* Same outputs
* Same file contents

Byte-for-byte reproducibility is the standard.

---

## 4.4 Partial Update Behavior

See Section 9.2 for the authoritative definition of partial update behavior. The lockfile supports partial updates: individual source entries may be updated independently while others remain unchanged.

---

# 5. Source Types

---

## 5.1 Git Source

Example:

```yaml
type: git
repo: https://github.com/org/repo.git
ref: v1.2.0
paths:
  - folder/
```

---

### Resolution Rules

agent-sync MUST resolve and record:

* commit SHA
* tree SHA
* per-file content hashes

`ref` is a human hint only.

The resolved commit SHA is authoritative.

---

### Optional Security Modes (Recommended)

Implementations SHOULD support:

* signed tag verification
* signed commit verification
* repository allowlists

---

## 5.2 URL Source

Example:

```yaml
type: url
url: https://example.com/file.md
checksum: sha256:abcdef
```

`checksum` is REQUIRED.

agent-sync MUST verify the fetched content against the declared checksum before accepting it.

---

## 5.3 Local Source

Example:

```yaml
type: local
path: ./agents/
```

### Path Resolution

Local source `path` is resolved relative to the project root (the directory containing `agent-sync.yaml`).

### Locking Requirements

Local sources MUST be locked by hashing all resolved files.

The lockfile MUST record per-file content hashes for local sources, identical in structure to git and url sources.

Rationale: local sources that are not hashed break the determinism guarantee. If the lockfile cannot verify local file integrity, drift detection and reproducibility are compromised.

---

# 6. Transform Specification

Transforms MUST be deterministic.

---

## 6.1 Template Transform

Variables substituted using Go `text/template`.

Example:

```
Project: {{ .project }}
```

---

### Security

Template execution:

* No file access
* No network access
* No code execution

Template input is data only.

Implementations SHOULD support disabling templates entirely via configuration.

---

## 6.2 Overrides

Overrides modify target files after sync.

Supported strategies:

* `append` — add content after the synced file
* `prepend` — add content before the synced file
* `replace` — replace the synced file entirely with the override file

Example:

```yaml
overrides:
  - target: security.md
    strategy: append
    file: local/security-extension.md
```

Override resolution:

1. Overrides are applied after all sources are synced.
2. The `target` field matches against the final destination filename (not source path).
3. If the target file does not exist after sync, the override MUST error.
4. The `file` path is resolved relative to the project root (the directory containing `agent-sync.yaml`).
5. The override file itself MUST exist at config validation time — agent-sync MUST error if it is not found.

---

## 6.3 Custom Transform Hooks

For transforms beyond template substitution and overrides, agent-sync supports custom transform hooks.

```yaml
transforms:
  - source: team-standards
    type: custom
    command: "./scripts/merge-yaml.sh"
    output_hash: sha256:expected...
```

### Custom Transform Rules

1. The command receives the source files via stdin or a temporary directory (implementation-defined).
2. The command MUST produce output to stdout or a temporary output directory (implementation-defined).
3. `output_hash` is OPTIONAL. If provided, agent-sync MUST verify the transform output against it. This enables lockfile-level pinning of transform results.
4. Custom transforms MUST NOT have network access.
5. Custom transforms MUST NOT modify files outside the designated output.
6. If the command exits non-zero, the sync MUST fail for that source.

### MVP Scope Note

For initial implementation, custom transforms are OPTIONAL. Template substitution and overrides cover the majority of use cases. Custom transforms provide the extension point for advanced workflows without overcomplicating the core.

---

## 6.4 Conflict Rules

Multiple sources targeting the same destination file MUST error unless:

* An explicit override is configured for that file, OR
* The target entries are for different tools (and thus resolve to different paths)

---

# 7. Target Specification

Targets define output locations.

---

## 7.1 Tool-Based Targets

```yaml
targets:
  - source: rules
    tools: [cursor, claude-code]
```

Resolves via the tool map (Section 3.2).

---

## 7.2 Explicit Path Targets

```yaml
targets:
  - source: rules
    destination: .custom/rules/
```

Used for tools without a built-in mapping, or when direct control is needed.

---

## 7.3 Target Rules

1. `tools` and `destination` are mutually exclusive per target entry.
2. All resolved paths MUST be validated against the sandbox (Section 8.1).
3. If a source defined in config has no matching target entry, agent-sync MUST warn during `sync` and `check`. This is not a fatal error — the source is simply unused.

---

# 8. Security Model

Security is a core design requirement.

---

## 8.1 Sandbox Guarantee

agent-sync MUST prevent writing outside the project root.

Protection MUST include:

* Absolute path resolution before any write
* Symlink resolution and traversal prevention
* TOCTOU (time-of-check-to-time-of-use) defense via atomic operations
* Rejection of paths containing `..` after resolution

`filepath.Clean` alone is NOT sufficient.

---

## 8.2 Safe Write Requirements

All file writes MUST:

* Use atomic temp file + rename
* Resolve the real path before writing
* Reject symlink escapes
* Verify the final path is within the project root

---

## 8.3 Integrity Verification

agent-sync MUST verify:

* File content hashes against lockfile
* Lockfile internal consistency (no duplicate entries, no orphaned references)

Verification MUST occur before any file is written to the target.

---

## 8.4 Cache Requirements

Cache MUST be:

* Content-addressed
* Immutable once written
* Verified (hash-checked) before use

---

## 8.5 Resource Limits (Recommended)

Implementations SHOULD support configurable limits:

* `max_file_size` — maximum size of any single fetched file
* `max_source_size` — maximum total size of files from a single source
* `fetch_timeout` — maximum time for any network fetch operation

---

# 9. CLI Specification

Reference CLI commands. The CLI is optional — all behavior is also available via the Go library (Section 10).

---

## 9.1 sync

Synchronize files to targets using the lockfile.

```
agent-sync sync [--dry-run]
```

Behavior:

* Reads the lockfile as the source of truth.
* Fetches content from cache or sources as needed.
* Writes files to target locations.
* MUST NOT modify the lockfile. (Only `update` and `prune` may modify the lockfile.)

`--dry-run` flag:

* Shows exactly which files would be written, modified, or removed.
* Shows the diff for each changed file.
* Makes no changes to the filesystem.
* Exits with the same code that a real sync would (0 for clean, non-zero for errors).

### Sync Failure and Rollback

If sync fails partway through:

* Files already written in the current operation MUST be rolled back to their previous state.
* The lockfile MUST NOT be modified.
* The CLI MUST report exactly which files were affected and which source caused the failure.
* Exit non-zero.

Rollback strategy: agent-sync SHOULD snapshot existing target files before beginning a sync operation and restore them on failure.

---

## 9.2 update

Resolve sources against their upstream and update the lockfile.

```
agent-sync update [source-name...] [--dry-run]
```

Behavior:

* Resolves each source to its current upstream state.
* Shows a diff of lockfile changes before applying.
* Requires interactive confirmation (unless `--yes` is passed).
* Updates the lockfile with resolved state.

If `source-name` arguments are provided, only those sources are updated. Others are left unchanged.

### Partial Failure Behavior

When updating multiple sources:

* Successfully resolved sources MUST be written to the lockfile.
* Failed sources MUST retain their previous lockfile entry.
* Each failure MUST be reported with source name and error detail.
* Exit non-zero if any source failed.

---

## 9.3 check

Verify that target files match the lockfile.

```
agent-sync check
```

Behavior:

* Hashes all target files and compares against the lockfile.
* Reports any drift (files changed, missing, or unexpected).
* Exit 0 if everything matches. Exit non-zero on drift.
* Suitable for CI pipelines.

---

## 9.4 verify

Verify the lockfile against upstream sources.

```
agent-sync verify [source-name...]
```

Behavior:

* For each source, checks whether the upstream has changed since the lockfile was last written.
* Reports which sources have newer content available.
* Does NOT modify the lockfile or target files.
* Exit 0 if all sources match upstream. Exit non-zero if any source has upstream changes.

### Use Cases

* CI pipeline: "are we up to date with upstream?"
* Security audit: "has the upstream content changed since we pinned it?"
* Pre-update check: "what would change if I ran update?"

---

## 9.5 status

Show the current state of all synced agents/sources.

```
agent-sync status [source-name...]
```

Output includes:

* Source name and type
* Pinned version / commit (from lockfile)
* Target destinations (resolved tool paths)
* Sync state: `synced`, `drifted`, `missing`, `pending`
* Last sync timestamp (if tracked)

Example output:

```
SOURCE           TYPE   PINNED AT       TARGETS                          STATE
base-rules       git    v1.2.0 (3f8c9a) .cursor/rules/, .claude/        synced
security-policy  url    sha256:abcdef   .cursor/policy.md, .claude/...   drifted
team-standards   local  (5 files)       .custom/agents/                  synced
```

---

## 9.6 info

Show information about the agent-sync tool itself.

```
agent-sync info
```

Output includes:

* agent-sync version
* Spec version supported
* Configuration file location (if found)
* Lockfile location (if found)
* Cache directory location and size
* Known tool definitions (built-in + custom)
* Go library version (if relevant)

---

## 9.7 prune

Remove previously synced files that are no longer referenced in the configuration.

```
agent-sync prune [--dry-run]
```

Behavior:

* Compares current config targets against files tracked in the lockfile.
* Removes files that were previously synced but are no longer in the config.
* Updates the lockfile to remove pruned entries.
* `--dry-run` shows what would be removed without acting.

---

## 9.8 Global Flags

The following flags are available on all commands:

* `--config <path>` — path to config file (default: `agent-sync.yaml`)
* `--lockfile <path>` — path to lockfile (default: `agent-sync.lock`)
* `--verbose` — detailed output
* `--quiet` — minimal output (errors only)
* `--no-color` — disable colored output
* `--no-inherit` — disable hierarchical config resolution (use only the project config)

---

## 9.9 Environment Variables

| Variable | Purpose |
|----------|---------|
| `AGENT_SYNC_SYSTEM_CONFIG` | Override the system config file path |
| `AGENT_SYNC_USER_CONFIG` | Override the user config file path |
| `AGENT_SYNC_NO_INHERIT` | Set to `1` or `true` to disable hierarchical resolution |

---

# 10. Library Specification (Go)

agent-sync MUST be usable as a Go library.

---

## 10.1 Core Interfaces

```go
type Resolver interface {
    // Resolve resolves sources and returns an updated lockfile.
    // If sourceNames is non-empty, only those sources are resolved.
    Resolve(ctx context.Context, config Config, sourceNames []string) (Lockfile, error)
}

type Syncer interface {
    Sync(ctx context.Context, lock Lockfile, config Config, opts SyncOptions) (*SyncResult, error)
}

type Checker interface {
    Check(ctx context.Context, lock Lockfile, config Config) (*CheckResult, error)
}

type Verifier interface {
    // Verify checks whether upstream sources have changed since lockfile was written.
    Verify(ctx context.Context, lock Lockfile, config Config, sourceNames []string) (*VerifyResult, error)
}

type Pruner interface {
    Prune(ctx context.Context, lock Lockfile, config Config, opts PruneOptions) (*PruneResult, error)
}
```

---

## 10.2 Result Types

```go
type SyncResult struct {
    Written  []FileAction
    Skipped  []FileAction
    Errors   []SourceError
}

type CheckResult struct {
    Clean   bool
    Drifted []DriftEntry
    Missing []string
}

type VerifyResult struct {
    UpToDate []string
    Changed  []SourceDelta
    Errors   []SourceError
}

type SyncOptions struct {
    DryRun bool
}

type PruneOptions struct {
    DryRun bool
}

type PruneResult struct {
    Removed []FileAction
    Errors  []SourceError
}
```

---

## 10.3 Library Rules

* Library MUST NOT depend on the CLI.
* Library MUST accept `context.Context` for cancellation and timeout.
* Library MUST return structured errors, not exit codes.
* All I/O MUST go through interfaces (filesystem, network) to enable testing and embedding.

---

# 11. Determinism Guarantees

Given:

* Byte-for-byte identical configuration file
* Byte-for-byte identical lockfile
* Same local source files (verified by lockfile hashes)
* Same override files

agent-sync MUST produce byte-for-byte identical output.

This applies to all operations: sync, transforms, and overrides.

Non-deterministic behavior is a bug.

---

# 12. Deferred Features

---

## 12.1 Install Command (Deferred)

Future:

```
agent-sync install org/security-agent
```

Would:

* Resolve a registry
* Fetch agent configuration
* Add source entry to config

Registry specification is deferred.

---

## 12.2 Registry Specification (Deferred)

agent-sync core remains registry-agnostic.

Registries MUST NOT be required for core functionality.

---

# 13. Error Handling

agent-sync MUST fail on:

* Hash mismatch (source, lockfile, or transform output)
* Sandbox violation
* Source integrity failure
* Ambiguous file writes (multiple sources, no override)
* Unknown tool name (no built-in or custom definition)
* Lockfile corruption or schema mismatch

agent-sync MUST NOT silently continue past errors.

All errors MUST include:

* The source or file that caused the failure
* The specific validation that failed
* A suggested remediation where possible

---

# 14. Versioning

## 14.1 Spec Version

The specification version is declared in both config and lockfile:

```yaml
version: 1
```

## 14.2 Version Semantics

* The `version` field is an integer.
* Each version defines a strict schema. Config and lockfile files MUST validate against the schema for their declared version.
* agent-sync MUST reject files with an unrecognized version.

## 14.3 Compatibility Rules

* **Patch changes** (new optional fields, clarifications): no version bump. Implementations MUST ignore unknown fields gracefully.
* **Breaking changes** (removed fields, changed semantics, new required fields): MUST increment the version integer.
* When a breaking version change occurs, agent-sync MUST provide a migration path — either an automatic `migrate` command or clear documentation of manual steps.
* agent-sync SHOULD support reading the immediately prior version and auto-migrating.

---

# 15. Summary

agent-sync provides:

* Deterministic file sync from git, URL, and local sources
* Secure sandboxed writes with symlink and TOCTOU protection
* Immutable lockfile with per-file content hashes
* Tool map for automatic multi-tool path resolution
* Partial failure resilience with rollback and granular reporting
* Registry-agnostic core with deferred install/registry
* Embeddable Go library with clean interfaces
* CLI with sync, update, check, verify, status, info, and prune commands

agent-sync is a synchronization engine, not an agent platform.

---

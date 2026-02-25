# Quick Start

This guide walks through setting up agent-sync in a project.

## 1. Initialize Configuration

```bash
agent-sync init
```

This creates an `agent-sync.yaml` with a commented starter template. Open it and edit the source to point at your repository:

```yaml
version: 1

sources:
  - name: team-rules
    type: git
    repo: https://github.com/your-org/agent-rules.git
    ref: main

targets:
  - source: team-rules
    tools: [cursor, claude-code]
```

This configuration will sync files from the `agent-rules` repository into both `.cursor/rules/` and `.claude/` directories.

## 2. Resolve and Lock

```bash
agent-sync update
```

This resolves the source to its current commit SHA, fetches file content hashes, and writes `agent-sync.lock`. The lockfile pins the exact state of every file.

## 3. Sync Files

```bash
agent-sync sync
```

This writes the locked files to the target directories. The lockfile is the source of truth — sync never modifies it.

## 4. Verify in CI

Add to your CI pipeline:

```bash
agent-sync check
```

This exits 0 if all target files match the lockfile, or non-zero if any files have drifted. Use this to enforce that synced files haven't been manually edited.

## Common Workflows

### Update a Single Source

```bash
agent-sync update team-rules
```

### Preview Changes

```bash
agent-sync sync --dry-run
agent-sync update --dry-run
```

### Check for Upstream Changes

```bash
agent-sync verify
```

Reports whether upstream sources have changed since the lockfile was written, without modifying anything.

### View Current State

```bash
agent-sync status
```

Shows each source's type, pinned version, targets, and sync state.

## Using Local Sources

For project-specific rules that live alongside your code:

```yaml
sources:
  - name: local-rules
    type: local
    path: ./agents/rules/

targets:
  - source: local-rules
    tools: [cursor]
```

Local sources are still locked by content hash, ensuring drift detection works.

## Using URL Sources

For single files:

```yaml
sources:
  - name: security-policy
    type: url
    url: https://example.com/security.md
    checksum: sha256:abc123...

targets:
  - source: security-policy
    destination: .cursor/rules/
```

URL sources require a `checksum` field for integrity verification.

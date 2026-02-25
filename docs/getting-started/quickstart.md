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

!!! tip "CI Tip"
    Use `--no-inherit` in CI environments to ignore system/user configs and ensure reproducible builds:
    ```bash
    agent-sync sync --no-inherit
    agent-sync check --no-inherit
    ```

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

## Hierarchical Configuration

agent-sync supports three config levels that merge together: system, user, and project. This lets you define global skill sets once and add project-specific sources where needed.

### User Config (applies to all projects)

Place at `~/Library/Application Support/agent-sync/agent-sync.yaml` (macOS) or `~/.config/agent-sync/agent-sync.yaml` (Linux):

```yaml
version: 1

sources:
  # Official Anthropic skills (mcp-builder, pdf, skill-creator, etc.)
  - name: anthropic-skills
    type: git
    repo: https://github.com/anthropics/skills.git
    ref: main
    paths:
      - skills/

  # Community skills
  - name: composio-skills
    type: git
    repo: https://github.com/ComposioHQ/awesome-claude-skills.git
    ref: master

targets:
  - source: anthropic-skills
    tools: [claude-code]

  - source: composio-skills
    tools: [claude-code]
```

### Project Config (project-specific additions)

In a specific project, add only the sources relevant to that codebase. The global skills are inherited automatically:

```yaml
version: 1

sources:
  # Security-focused PR/diff review
  - name: tob-differential-review
    type: git
    repo: https://github.com/trailofbits/skills.git
    ref: main
    paths:
      - plugins/differential-review/

  # Property-based testing guidance
  - name: tob-property-testing
    type: git
    repo: https://github.com/trailofbits/skills.git
    ref: main
    paths:
      - plugins/property-based-testing/

targets:
  - source: tob-differential-review
    tools: [claude-code]

  - source: tob-property-testing
    tools: [claude-code]
```

Running `agent-sync info` shows the full config chain:

```
$ agent-sync info
agent-sync dev
  config chain:
    system:    /etc/agent-sync/agent-sync.yaml (not found)
    user:      ~/Library/Application Support/agent-sync/agent-sync.yaml (loaded)
    project:   agent-sync.yaml (loaded)
```

The merged result includes all sources from both levels. See the [Enterprise Configuration](../guides/enterprise-config.md) guide for advanced patterns.

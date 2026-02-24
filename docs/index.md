# agent-sync

**Deterministic, registry-agnostic synchronization of agent files.**

agent-sync fetches files from external sources (Git repositories, URLs, local paths), pins them immutably via a lockfile, applies controlled transformations, and synchronizes them into tool-specific project locations.

## Key Features

- **Deterministic**: Same inputs always produce byte-for-byte identical outputs
- **Source types**: Git repositories, URLs with checksum verification, local paths
- **Tool map**: Built-in support for Cursor, Claude Code, Copilot, Windsurf, Cline, and Codex
- **Lockfile pinning**: Immutable content hashes ensure reproducibility
- **Transforms**: Template variable substitution and file overrides
- **Security**: Sandbox enforcement, atomic writes, symlink traversal prevention
- **CI-friendly**: `check` and `verify` commands with structured exit codes

## Quick Example

```yaml
# agent-sync.yaml
version: 1

sources:
  - name: team-rules
    type: git
    repo: https://github.com/org/agent-rules.git
    ref: v1.0.0

targets:
  - source: team-rules
    tools: [cursor, claude-code]
```

```bash
# Resolve sources and create lockfile
agent-sync update

# Sync files to tool directories
agent-sync sync

# Verify nothing has drifted
agent-sync check
```

## Next Steps

- [Installation](getting-started/installation.md) — Install the CLI
- [Quick Start](getting-started/quickstart.md) — Set up your first project
- [Configuration Reference](reference/config.md) — Full config file documentation
- [CLI Reference](reference/cli.md) — All available commands

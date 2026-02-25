# agent-sync

[![CI](https://github.com/bianoble/agent-sync/actions/workflows/ci.yml/badge.svg)](https://github.com/bianoble/agent-sync/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/bianoble/agent-sync.svg)](https://pkg.go.dev/github.com/bianoble/agent-sync)
[![Go Report Card](https://goreportcard.com/badge/github.com/bianoble/agent-sync)](https://goreportcard.com/report/github.com/bianoble/agent-sync)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Deterministic, registry-agnostic synchronization of agent files.**

agent-sync fetches files from external sources (Git repos, URLs, local paths), pins them immutably via a lockfile, applies controlled transformations, and synchronizes them into tool-specific project locations. One config, every tool, every repo.

## Features

- **Multi-tool support** — Built-in mappings for Cursor, Claude Code, Copilot, Windsurf, Cline, and Codex
- **Deterministic** — Same inputs always produce byte-for-byte identical outputs
- **Lockfile pinning** — Content-addressed SHA256 hashes ensure reproducibility and supply chain integrity
- **Hierarchical config** — System, user, and project configs merge automatically; override per-project as needed
- **Transforms** — Template variable substitution and file overrides (append, prepend, replace)
- **Secure by default** — Sandbox enforcement, atomic writes, symlink traversal prevention

## Install

```bash
# macOS / Linux
curl -sSL https://raw.githubusercontent.com/bianoble/agent-sync/main/install.sh | sh

# Go
go install github.com/bianoble/agent-sync/cmd/agent-sync@latest
```

## Quick Start

```bash
# 1. Create a starter config
agent-sync init

# 2. Edit agent-sync.yaml to point at your sources, then resolve and lock
agent-sync update

# 3. Sync files to tool directories
agent-sync sync

# 4. Verify nothing has drifted (CI-friendly)
agent-sync check
```

Example `agent-sync.yaml`:

```yaml
version: 1

sources:
  - name: team-rules
    type: git
    repo: https://github.com/org/agent-rules.git
    ref: v1.0.0

targets:
  - source: team-rules
    tools: [cursor, claude-code, copilot]
```

## Documentation

- [Installation](https://bianoble.github.io/agent-sync/getting-started/installation/) — Install scripts, pre-built binaries, and build from source
- [Quick Start](https://bianoble.github.io/agent-sync/getting-started/quickstart/) — Set up your first project
- [Configuration Reference](https://bianoble.github.io/agent-sync/reference/config/) — Full config file documentation
- [CLI Reference](https://bianoble.github.io/agent-sync/reference/cli/) — All available commands
- [Go Library](https://bianoble.github.io/agent-sync/reference/library/) — Use agent-sync as a Go library
- [Enterprise & DevSecOps](https://bianoble.github.io/agent-sync/guides/enterprise-config/) — Hierarchical config, compliance, and deployment at scale

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, and PR guidelines.

## License

[MIT](LICENSE)

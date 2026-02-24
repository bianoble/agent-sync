# Lockfile Reference

The `agent-sync.lock` file records the fully resolved, immutable state of all sources. It is the single source of truth for `sync` and `check` operations.

## Structure

```yaml
version: 1
sources:
  - name: team-rules
    type: git
    repo: https://github.com/org/repo.git
    resolved:
      commit: abc123...
      tree: def456...
      files:
        rules/general.md:
          sha256: 789abc...
        rules/security.md:
          sha256: 012def...
    status: ok
```

## Fields

### Top Level

| Field     | Type | Description |
|-----------|------|-------------|
| `version` | int  | Must be `1` |
| `sources` | list | Resolved source entries |

### Source Entry

| Field      | Type   | Description |
|------------|--------|-------------|
| `name`     | string | Source identifier (matches config) |
| `type`     | string | `git`, `url`, or `local` |
| `repo`     | string | Repository URL (git only) |
| `resolved` | object | Type-specific resolved state |
| `status`   | string | Resolution status |

### Resolved State (Git)

| Field    | Type   | Description |
|----------|--------|-------------|
| `commit` | string | Full commit SHA |
| `tree`   | string | Tree SHA |
| `files`  | map    | Relative path to file hash |

### Resolved State (URL)

| Field    | Type   | Description |
|----------|--------|-------------|
| `url`    | string | Fetched URL |
| `sha256` | string | Content hash |
| `files`  | map    | Relative path to file hash |

### Resolved State (Local)

| Field   | Type   | Description |
|---------|--------|-------------|
| `path`  | string | Resolved path |
| `files` | map    | Relative path to file hash |

## Rules

- Only `update` and `prune` may modify the lockfile
- `sync` reads the lockfile but never writes to it
- Per-file SHA256 hashes enable drift detection and cache lookup
- The resolved commit SHA (not the config `ref`) is authoritative for git sources
- Duplicate source entries are not allowed

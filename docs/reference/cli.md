# CLI Reference

## Global Flags

These flags are available on all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `--config <path>` | `agent-sync.yaml` | Path to config file |
| `--lockfile <path>` | `agent-sync.lock` | Path to lockfile |
| `--verbose` | `false` | Detailed output |
| `--quiet` | `false` | Minimal output (errors only) |
| `--no-color` | `false` | Disable colored output |

## Commands

### sync

Synchronize files to targets using the lockfile.

```bash
agent-sync sync [--dry-run]
```

- Reads the lockfile as the source of truth
- Fetches content from cache or sources as needed
- Writes files to target locations
- Does **not** modify the lockfile

**Flags:**

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would change without writing files |

**Rollback:** If sync fails partway through, files already written are rolled back to their previous state.

---

### update

Resolve sources against upstream and update the lockfile.

```bash
agent-sync update [source-name...] [--dry-run] [--yes]
```

- Resolves each source to its current upstream state
- Shows a diff of lockfile changes before applying
- Requires interactive confirmation (unless `--yes`)
- Updates the lockfile with resolved state

If source names are provided, only those sources are updated.

**Flags:**

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would change without updating the lockfile |
| `--yes` | Skip interactive confirmation |

**Partial failure:** Successfully resolved sources are written; failed sources retain their previous lockfile entry. Exit non-zero if any failed.

---

### check

Verify that target files match the lockfile.

```bash
agent-sync check
```

- Hashes all target files and compares against the lockfile
- Reports any drift (files changed, missing, or unexpected)
- Exit 0 if everything matches; exit non-zero on drift

Suitable for CI pipelines.

---

### verify

Verify the lockfile against upstream sources.

```bash
agent-sync verify [source-name...]
```

- Checks whether upstream has changed since the lockfile was written
- Reports which sources have newer content available
- Does **not** modify the lockfile or target files
- Exit 0 if all match; exit non-zero if changes are available

---

### status

Show the current state of all synced sources.

```bash
agent-sync status [source-name...]
```

Output columns:

| Column | Description |
|--------|-------------|
| SOURCE | Source name |
| TYPE | `git`, `url`, or `local` |
| PINNED AT | Lockfile version/commit |
| TARGETS | Resolved destination paths |
| STATE | `synced`, `drifted`, `missing`, `pending` |

---

### info

Show information about the agent-sync installation.

```bash
agent-sync info
```

Displays version, spec version, config/lockfile paths, cache directory and size, and known tool definitions.

---

### prune

Remove files no longer referenced in the configuration.

```bash
agent-sync prune [--dry-run]
```

- Compares current config targets against lockfile-tracked files
- Removes files that were previously synced but are no longer in the config

**Flags:**

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would be removed without acting |

---

### version

Print version information.

```bash
agent-sync version
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error or drift detected |

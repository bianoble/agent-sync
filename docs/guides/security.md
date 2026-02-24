# Security Model

Security is a core design requirement of agent-sync.

## Sandbox Guarantee

agent-sync prevents writing outside the project root. Protection includes:

- **Absolute path resolution** before any write
- **Symlink resolution** and traversal prevention
- **TOCTOU defense** via atomic operations
- **Path validation** — paths containing `..` are rejected after resolution

`filepath.Clean` alone is not sufficient — agent-sync resolves the real path through symlinks.

## Safe Write

All file writes use atomic operations:

1. Write content to a temporary file in the target directory
2. Sync the temporary file to disk (`fsync`)
3. Rename the temporary file to the final path (atomic on POSIX)
4. Verify the final path is within the project root

This prevents partial writes and ensures file integrity.

## Content Integrity

### Lockfile Hashing

Every file is tracked by its SHA256 content hash in the lockfile. This enables:

- **Drift detection**: `check` compares target file hashes against the lockfile
- **Cache verification**: Cached content is re-verified before use
- **Determinism**: Same lockfile always produces identical output

### URL Checksum Verification

URL sources require a `checksum` field:

```yaml
sources:
  - name: policy
    type: url
    url: https://example.com/file.md
    checksum: sha256:abcdef...
```

Fetched content is verified against the declared checksum before acceptance.

### Cache Integrity

The content-addressed cache:

- Stores files by their SHA256 hash
- Is immutable once written
- Re-verifies hashes on retrieval (self-healing on corruption)

## Rollback on Failure

If `sync` fails partway through:

1. Files already written are rolled back to their previous state
2. The lockfile is never modified
3. The error report includes which files were affected and which source caused the failure

## Template Security

Template transforms are sandboxed:

- No file access
- No network access
- No code execution
- Missing variables produce errors (not empty strings)

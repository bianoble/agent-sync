# Transforms

Transforms modify file content during sync. All transforms are deterministic — the same inputs always produce the same outputs.

## Template Transform

Substitutes variables using Go `text/template` syntax.

### Configuration

```yaml
variables:
  project: my-app
  team: platform

transforms:
  - source: team-rules
    type: template
    vars:
      env: production
```

### Template Syntax

In source files:

```
Project: {{ .project }}
Team: {{ .team }}
Environment: {{ .env }}
```

Variables are merged: per-transform `vars` override global `variables`.

### Security

Template execution is sandboxed:

- No file access
- No network access
- No code execution
- Template input is data only

Binary files (containing null bytes or non-UTF8 content) are skipped.

## Overrides

Overrides modify target files **after** all sources are synced.

### Strategies

| Strategy | Behavior |
|----------|----------|
| `append` | Add content after the synced file |
| `prepend` | Add content before the synced file |
| `replace` | Replace the synced file entirely |

### Configuration

```yaml
overrides:
  - target: security.md
    strategy: append
    file: local/security-extension.md
```

### Rules

1. Overrides are applied after all sources are synced
2. `target` matches against the final destination filename (not source path)
3. The target file must exist after sync — otherwise it's an error
4. `file` is resolved relative to the project root
5. The override file must exist at config validation time

### Example: Appending Local Rules

```yaml
# Sync shared security policy, then append team-specific additions
sources:
  - name: security
    type: git
    repo: https://github.com/org/policies.git
    ref: v2.0

targets:
  - source: security
    tools: [cursor]

overrides:
  - target: policy.md
    strategy: append
    file: local/team-policy-additions.md
```

## Conflict Rules

Multiple sources targeting the same destination file produce an error unless:

- An explicit override is configured for that file, **or**
- The target entries are for different tools (and thus resolve to different paths)

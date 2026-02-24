# Configuration Reference

The `agent-sync.yaml` file defines sources, targets, transforms, and overrides.

## Top-Level Structure

```yaml
version: 1

sources:
  - name: ...
    type: git | url | local
    # type-specific fields

targets:
  - source: ...
    tools: [...]     # OR
    destination: ... # mutually exclusive with tools

variables:
  key: value

transforms:
  - source: ...
    type: template
    vars:
      key: value

overrides:
  - target: filename
    strategy: append | prepend | replace
    file: path/to/override

tool_definitions:
  - name: tool-name
    destination: .tool/path/
```

## Sources

### Git Source

```yaml
sources:
  - name: team-rules
    type: git
    repo: https://github.com/org/repo.git
    ref: v1.0.0
    paths:
      - rules/
```

| Field  | Required | Description |
|--------|----------|-------------|
| `name` | Yes | Unique identifier for this source |
| `type` | Yes | Must be `git` |
| `repo` | Yes | Git repository URL |
| `ref`  | Yes | Branch, tag, or commit (human hint; resolved commit SHA is authoritative) |
| `paths` | No | Filter to specific paths within the repo |

### URL Source

```yaml
sources:
  - name: policy
    type: url
    url: https://example.com/file.md
    checksum: sha256:abcdef...
```

| Field      | Required | Description |
|------------|----------|-------------|
| `name`     | Yes | Unique identifier |
| `type`     | Yes | Must be `url` |
| `url`      | Yes | HTTPS URL to fetch |
| `checksum` | Yes | `sha256:<hex>` checksum for integrity verification |

### Local Source

```yaml
sources:
  - name: local-rules
    type: local
    path: ./agents/rules/
```

| Field  | Required | Description |
|--------|----------|-------------|
| `name` | Yes | Unique identifier |
| `type` | Yes | Must be `local` |
| `path` | Yes | Path relative to the project root |

## Targets

Each target maps a source to one or more destinations.

### Tool-Based Targets

```yaml
targets:
  - source: team-rules
    tools: [cursor, claude-code, copilot]
```

Tool names are resolved via the [tool map](../guides/toolmap.md).

### Explicit Path Targets

```yaml
targets:
  - source: team-rules
    destination: .custom/agents/
```

!!! warning
    `tools` and `destination` are mutually exclusive on a single target entry.

## Variables

Global variables available to template transforms:

```yaml
variables:
  project: my-app
  team: platform
```

## Transforms

See the [Transforms Guide](../guides/transforms.md) for details.

```yaml
transforms:
  - source: team-rules
    type: template
    vars:
      project: my-app
```

## Overrides

See the [Transforms Guide](../guides/transforms.md#overrides) for details.

```yaml
overrides:
  - target: security.md
    strategy: append
    file: local/security-extension.md
```

## Tool Definitions

Custom tool definitions override or extend built-in mappings:

```yaml
tool_definitions:
  - name: my-tool
    destination: .my-tool/config/
```

## Validation Rules

- `version` must be `1`
- Source names must be unique
- Each source type requires its specific fields
- `tools` and `destination` are mutually exclusive per target
- Override `strategy` must be `append`, `prepend`, or `replace`
- Override `file` must exist at validation time
- Unknown fields are ignored (forward compatibility)

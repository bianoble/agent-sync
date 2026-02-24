# Tool Map

The tool map resolves tool names in target definitions to filesystem paths.

## Built-in Tools

| Tool | Destination |
|------|-------------|
| `cursor` | `.cursor/rules/` |
| `claude-code` | `.claude/` |
| `copilot` | `.github/copilot/` |
| `windsurf` | `.windsurf/rules/` |
| `cline` | `.cline/rules/` |
| `codex` | `.codex/` |

## Using Tools in Targets

```yaml
targets:
  - source: team-rules
    tools: [cursor, claude-code]
```

This writes files from `team-rules` to both `.cursor/rules/` and `.claude/`.

## Custom Tool Definitions

Define custom tools to extend or override the built-in mappings:

```yaml
tool_definitions:
  - name: my-agent
    destination: .my-agent/config/
```

Custom definitions override built-in ones if the name matches:

```yaml
tool_definitions:
  - name: cursor
    destination: .cursor/custom-path/
```

## Resolution Rules

1. `tools:` on a target resolves each tool name to its destination path
2. Custom definitions take precedence over built-in ones
3. Unknown tool names produce an error with guidance on how to define it
4. `destination:` on a target is used as-is (no tool map lookup)
5. `tools:` and `destination:` are mutually exclusive per target entry

## Explicit Paths

For tools without built-in support, use `destination:` directly:

```yaml
targets:
  - source: team-rules
    destination: .custom/agents/
```

# Enterprise Configuration for DevSecOps

agent-sync supports **hierarchical configuration resolution** across three levels — system, user, and project — enabling organizations to enforce security baselines, standardize AI agent behavior, and maintain compliance at scale.

This guide is written for **DevSecOps engineers, security teams, and platform engineers** responsible for managing AI coding tools across development teams.

## Overview

Configuration is loaded and merged from up to three locations, lowest precedence first:

| Level | Purpose | Managed by |
|-------|---------|------------|
| **System** | Organization-wide mandated policies | Platform / Security team |
| **User** | Per-developer preferences and defaults | Individual developer |
| **Project** | Project-specific sources and targets | Project maintainers |

Higher-precedence configs override lower ones. Project-level always wins.

## Config File Locations

### macOS

| Level | Path |
|-------|------|
| System | `/etc/agent-sync/agent-sync.yaml` |
| User | `~/Library/Application Support/agent-sync/agent-sync.yaml` |
| Project | `./agent-sync.yaml` |

### Linux

| Level | Path |
|-------|------|
| System | `/etc/agent-sync/agent-sync.yaml` |
| User | `$XDG_CONFIG_HOME/agent-sync/agent-sync.yaml` (default: `~/.config/agent-sync/agent-sync.yaml`) |
| Project | `./agent-sync.yaml` |

### Windows

| Level | Path |
|-------|------|
| System | `%ProgramData%\agent-sync\agent-sync.yaml` |
| User | `%AppData%\agent-sync\agent-sync.yaml` |
| Project | `.\agent-sync.yaml` |

## Use Case: Enforcing Security Baselines

A security team can deploy a system-level config that pins approved AI agent rule files to specific reviewed versions. Every project on that machine inherits these sources automatically.

### System Config (deployed via MDM/Ansible/Chef)

```yaml
# /etc/agent-sync/agent-sync.yaml
version: 1

variables:
  org: acme-corp
  compliance_level: soc2

sources:
  - name: org-security-rules
    type: git
    repo: https://github.com/acme-corp/ai-security-rules.git
    ref: v2.3.1
    paths:
      - policies/

  - name: org-approved-prompts
    type: url
    url: https://security.acme-corp.com/approved-prompts.md
    checksum: sha256:a1b2c3d4e5f6...

targets:
  - source: org-security-rules
    tools: [cursor, claude-code, copilot, windsurf]

  - source: org-approved-prompts
    tools: [cursor, claude-code]

overrides:
  - target: security.md
    strategy: prepend
    file: /etc/agent-sync/compliance-header.md
```

### Project Config (maintained by developers)

```yaml
# ./agent-sync.yaml
version: 1

variables:
  project: payment-service
  team: payments

sources:
  - name: team-rules
    type: local
    path: ./agents/rules/

targets:
  - source: team-rules
    tools: [cursor, claude-code]
```

### Merged Result

When a developer runs `agent-sync sync` in the project directory, the merged config includes:

- **Sources**: `org-security-rules` + `org-approved-prompts` (from system) + `team-rules` (from project)
- **Targets**: All targets from both levels
- **Variables**: `org=acme-corp`, `compliance_level=soc2`, `project=payment-service`, `team=payments`

The security team's rules are always present. Developers add project-specific rules alongside them.

## Merge Semantics

| Field | Strategy | Detail |
|-------|----------|--------|
| `version` | Must agree | Fatal error if system says `1` and project says `2` |
| `variables` | Deep merge | Higher-precedence keys overwrite; unique keys preserved |
| `sources` | Merge by name | Same `name` in project fully replaces system entry |
| `tool_definitions` | Merge by name | Same `name` in project replaces system entry |
| `targets` | Concatenate | System targets first, then user, then project |
| `overrides` | Concatenate | Applied in order: system, user, project |
| `transforms` | Concatenate | Applied in order: system, user, project |

!!! note "Source Override Visibility"
    If a project redefines a source with the same `name` as a system source, the project's definition completely replaces the system one. This is auditable — a code review of `agent-sync.yaml` shows exactly which org sources a project overrides.

## Deploying System Configs at Scale

### macOS (MDM Profiles)

Use Jamf, Mosyle, or Kandji to deploy a configuration profile that places the config file at `/etc/agent-sync/agent-sync.yaml`:

```bash
# Example install script
sudo mkdir -p /etc/agent-sync
sudo cp agent-sync.yaml /etc/agent-sync/agent-sync.yaml
sudo chmod 644 /etc/agent-sync/agent-sync.yaml
```

### Linux (Configuration Management)

=== "Ansible"

    ```yaml
    - name: Deploy agent-sync system config
      copy:
        src: files/agent-sync.yaml
        dest: /etc/agent-sync/agent-sync.yaml
        owner: root
        group: root
        mode: '0644'
    ```

=== "Chef"

    ```ruby
    directory '/etc/agent-sync' do
      owner 'root'
      group 'root'
      mode '0755'
    end

    cookbook_file '/etc/agent-sync/agent-sync.yaml' do
      source 'agent-sync.yaml'
      owner 'root'
      group 'root'
      mode '0644'
    end
    ```

### Windows (Group Policy / SCCM)

Deploy the config to `C:\ProgramData\agent-sync\agent-sync.yaml` using Group Policy file deployment or SCCM application packaging.

```powershell
# PowerShell deployment script
$dir = "$env:ProgramData\agent-sync"
New-Item -ItemType Directory -Force -Path $dir
Copy-Item agent-sync.yaml "$dir\agent-sync.yaml"
```

## CI/CD Integration

In CI environments, hierarchical resolution should be disabled for reproducible builds. Use the `--no-inherit` flag or the environment variable:

```yaml
# GitHub Actions
- name: Sync agent files
  run: agent-sync sync --no-inherit

# Or via environment variable
- name: Sync agent files
  env:
    AGENT_SYNC_NO_INHERIT: "1"
  run: agent-sync sync
```

This ensures CI uses only the project's `agent-sync.yaml` and is not affected by system or user configs on the runner.

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `AGENT_SYNC_SYSTEM_CONFIG` | Override the system config file path |
| `AGENT_SYNC_USER_CONFIG` | Override the user config file path |
| `AGENT_SYNC_NO_INHERIT` | Set to `1` or `true` to disable hierarchical resolution |

## Verifying the Config Chain

Use `agent-sync info` to see which config files are active:

```
$ agent-sync info
agent-sync v0.5.0
  spec version:  1
  config chain:
    system:    /etc/agent-sync/agent-sync.yaml (loaded)
    user:      /Users/alice/Library/Application Support/agent-sync/agent-sync.yaml (not found)
    project:   ./agent-sync.yaml (loaded)
  lockfile:      agent-sync.lock
  cache dir:     /Users/alice/.cache/agent-sync
  cache size:    2.1 MB
```

Use `--verbose` with any command to see detailed merge activity:

```
$ agent-sync sync --verbose
  config: loaded system /etc/agent-sync/agent-sync.yaml
  config: user /Users/alice/Library/Application Support/agent-sync/agent-sync.yaml (not found, skipping)
  config: loaded project ./agent-sync.yaml
  config: merged 2 layers
  ...
```

## Compliance Patterns

### Mandatory Security Rules

Deploy a system config with sources that reference your organization's approved AI agent rule files. Pin to specific reviewed tags (not branches) for auditability.

### Approved Tool Definitions

Use system-level `tool_definitions` to standardize which tools map to which directories:

```yaml
tool_definitions:
  - name: cursor
    destination: .cursor/rules/approved/
  - name: claude-code
    destination: .claude/approved/
```

### Audit Trail

The lockfile (`agent-sync.lock`) records the exact commit SHA, URL checksums, and file hashes for every synced source. This provides a verifiable audit trail of which agent rules were active at any point in time.

Combined with `agent-sync check` in CI, teams can detect and prevent drift from approved configurations:

```yaml
# CI step: fail if agent files have been manually modified
- name: Check for agent file drift
  run: agent-sync check
```

### Preventing Override of Mandated Sources

While projects can technically redefine a system source by using the same `name`, this is visible in code review. For stricter enforcement, consider:

1. **CI checks**: Validate that project configs don't redefine system source names
2. **Pre-commit hooks**: Reject changes that shadow system sources
3. **agent-sync check**: Run in CI to verify all expected sources are present with correct versions

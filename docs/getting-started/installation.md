# Installation

## Quick Install (macOS / Linux)

```bash
curl -sSL https://raw.githubusercontent.com/bianoble/agent-sync/main/install.sh | sh
```

This auto-detects your OS and architecture, downloads the latest release, verifies the SHA256 checksum, and installs to `/usr/local/bin` (or `~/.local/bin` if not writable).

To install a specific version:

```bash
curl -sSL https://raw.githubusercontent.com/bianoble/agent-sync/main/install.sh | VERSION=v0.1.0 sh
```

## Quick Install (Windows)

```powershell
irm https://raw.githubusercontent.com/bianoble/agent-sync/main/install.ps1 | iex
```

Downloads the latest release, verifies the SHA256 checksum, installs to `%LOCALAPPDATA%\agent-sync\bin`, and adds it to your PATH.

## Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/bianoble/agent-sync/releases).

Available platforms:

| OS      | Architecture | Binary                          |
|---------|--------------|---------------------------------|
| Linux   | amd64        | `agent-sync-linux-amd64`        |
| Linux   | arm64        | `agent-sync-linux-arm64`        |
| macOS   | amd64        | `agent-sync-darwin-amd64`       |
| macOS   | arm64        | `agent-sync-darwin-arm64`       |
| Windows | amd64        | `agent-sync-windows-amd64.exe`  |

## Go Install

With Go 1.24+:

```bash
go install github.com/bianoble/agent-sync/cmd/agent-sync@latest
```

## Build from Source

```bash
git clone https://github.com/bianoble/agent-sync.git
cd agent-sync
make build
```

The binary will be at `./agent-sync`.

## Verify Installation

```bash
agent-sync version
```

## Go Library

To use agent-sync as a Go library:

```bash
go get github.com/bianoble/agent-sync@latest
```

See the [Go Library Reference](../reference/library.md) for API documentation.

# Installation

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

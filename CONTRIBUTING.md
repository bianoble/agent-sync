# Contributing to agent-sync

Thank you for your interest in contributing to agent-sync! This document covers everything you need to get started.

## Prerequisites

- **Go 1.24+** — [download](https://go.dev/dl/)
- **golangci-lint** — `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- **git**

## Development Setup

```bash
git clone https://github.com/bianoble/agent-sync.git
cd agent-sync
make build
make test
```

## Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile binary with version/commit/date ldflags to `bin/agent-sync` |
| `make test` | Run all tests with race detector |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code with gofmt and goimports |
| `make vet` | Run `go vet` |
| `make clean` | Remove binaries and test cache |

## Running the Full CI Check Locally

Before submitting a PR, run the same checks CI runs:

```bash
make lint && make test && make vet
```

## Code Style

- **Formatting**: `gofmt` and `goimports` — run `make fmt` before committing
- **Linting**: golangci-lint must pass clean (see `.golangci.yml` for enabled linters)
- **Error checking**: Use `errors.Is(err, os.ErrNotExist)`, not `os.IsNotExist(err)` — the former correctly unwraps wrapped errors
- **Naming**: Follow standard Go conventions; exported names have doc comments

## Test Expectations

- All tests must pass with the race detector enabled (`go test -race ./...`)
- New code should include tests
- Use `t.TempDir()` for temporary directories (auto-cleaned)
- Use `t.Setenv()` for environment variable manipulation in tests (auto-restored)
- Prefer table-driven tests for functions with multiple input/output cases
- Tests that rely on filesystem permissions should skip when running as root: `if os.Getuid() == 0 { t.Skip("test unreliable as root") }`

## Pull Request Workflow

1. **Branch from `main`**: Create a feature branch with a descriptive name
2. **One concern per PR**: Keep changes focused — a single feature, fix, or refactor
3. **Write descriptive commits**: Use imperative mood, short first line (e.g., "Add lockfile validation for empty sources")
4. **Ensure CI passes**: Run `make lint && make test && make vet` before pushing
5. **Open a PR against `main`**: Include a summary of what changed and why

## Project Structure

```
cmd/agent-sync/          CLI entrypoint and Cobra commands
internal/
  cache/                 Content-addressed file cache (SHA256)
  config/                YAML config loading, hierarchical discovery, merge
  engine/                Core operations: sync, update, check, verify, status, prune
  lock/                  Lockfile read/write/validate
  sandbox/               Safe-write enforcement (path traversal, symlink prevention)
  source/                Source resolvers (git, url, local) and filesystem abstractions
  target/                Tool map resolution and target path computation
  transform/             Template and override transforms
pkg/agentsync/           Public Go library API
docs/                    MkDocs documentation site
```

## Questions?

Open an issue or start a discussion on GitHub. We're happy to help!

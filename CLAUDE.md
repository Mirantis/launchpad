# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Mirantis Launchpad is a CLI tool that installs, upgrades, and resets Mirantis Kubernetes Engine (MKE) and Mirantis Container Runtime (MCR) clusters on provisioned compute nodes. It is **stateless** between runs — all cluster state is discovered by querying hosts directly.

## Agent Rules (from AI_AGENTS.md)

- **NEVER** work on, push to, or merge into `main`. All work goes on feature branches.
- **All commits MUST be signed** (`git commit -s`).
- Use `GOTOOLCHAIN=auto` for all `go` commands (already set in Makefile via `export`).
- Read `docs/guidance/project.md` before implementing features or bug fixes.

## Commands

```bash
# Build
make local              # Build for current platform → dist/launchpad_GOOS_GOARCH

# Lint & security
make lint               # golangci-lint run
make security-scan      # govulncheck ./...

# Tests
make unit-test          # go test -v --tags 'testing' ./pkg/...
make functional-test    # go test -v ./test/functional/... -timeout 20m
make integration-test   # go test -v ./test/integration/... -timeout 20m
make smoke-small        # E2E small cluster (20m timeout)
make smoke-full         # E2E full matrix cluster (50m timeout)

# Run a single test package
go test -v --tags 'testing' ./pkg/config/...

# Run a single test
go test -v --tags 'testing' ./pkg/config/... -run TestFoo
```

The build tag `testing` is required for unit tests in `pkg/`.

## Architecture: Phase Manager Pattern

All major operations (apply, reset, describe) are implemented as ordered sequences of **phases** executed by a `phase.Manager`. This is the central architectural pattern — new features should be implemented as new phases or additions to existing ones.

```
cmd/apply.go
  └── product/mke/mke.go (Apply method)
        └── phase.Manager.Run()
              └── [Phase1, Phase2, ..., PhaseN] executed sequentially
```

Each phase implements:
- `Run() error` — required
- `Title() string` — required
- Optional: `Prepare(config)`, `ShouldRun()`, `CleanUp()`, `DisableCleanup()`

**Key packages:**

| Package | Role |
|---|---|
| `pkg/phase/` | Phase Manager orchestration |
| `pkg/product/mke/` | MKE apply/reset/describe logic |
| `pkg/product/mke/phase/` | 30+ phase implementations |
| `pkg/product/mke/config/` | MKE-specific config structs |
| `pkg/config/` | YAML config parsing, schema migrations (v1–v15) |
| `pkg/configurer/` | OS/distro-specific host configuration |
| `pkg/kubeclient/` | Kubernetes client operations |
| `pkg/helm/` | Helm chart management |
| `pkg/docker/` | Docker image handling and auth |
| `pkg/analytics/` | Segment telemetry |

## Configuration

- Default config file: `launchpad.yaml` (override with `--config`)
- Supports `--config -` to read from stdin
- Environment variable substitution via `envsubst`
- Schema migrations live in `pkg/config/migration/` — required for any config struct changes
- Config migrations must be backward compatible; each migration version has unit tests

## Host Management

Hosts are managed via [k0sproject/rig](https://github.com/k0sproject/rig), which abstracts SSH (Linux) and WinRM (Windows) connections. Phases receive a configured host set and use rig for remote command execution, file upload/download, and shell quoting.

## Testing Strategy

| Type | Location | Notes |
|---|---|---|
| Unit | `pkg/**/*_test.go` | Requires `--tags 'testing'` build tag |
| Functional | `test/functional/` | Component-level, may need network |
| Integration | `test/integration/` | Requires real provisioned nodes |
| Smoke | `test/smoke/` | Full E2E via Terraform (terratest) |

## Linting

`.golangci.yml` enables 30+ linters. Notable constraints:
- `varnamelen`: max 10 chars (allowlist includes `i`, `h`, `ok`, `id`, etc.)
- Package names may conflict with stdlib (log, version, user, constant) — these are excluded from the relevant linter
- Generated files (`*.gen.go`) are excluded

## Documentation

Consult these before implementing non-trivial changes:
- `docs/guidance/project.md` — core architectural principles
- `docs/specifications/architecture.md` — Phase Manager and design decisions
- `docs/development/workflow.md` — contribution and testing workflow
- `docs/requirements/` — PRDs for planned features

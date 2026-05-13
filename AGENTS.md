# Mirantis Launchpad — Agent Instructions

This file follows the [AGENTS.md](https://agents.md/) open standard and is read by Claude Code, Cursor, Windsurf, Codex, Gemini CLI, and compatible agents. Instructions here take precedence over general tool defaults.

---

## Project Overview

Launchpad is a Go CLI that installs, upgrades, and resets Mirantis Kubernetes Engine (MKE) and Mirantis Container Runtime (MCR) clusters on provisioned compute nodes. It is **stateless** between runs — all cluster state is discovered by querying hosts directly at the start of each operation.

---

## Non-Negotiable Rules

- **NEVER** commit to, push to, or merge into `main`. All work goes on feature branches.
- **All commits MUST be signed**: `git commit -s`.
- Use `GOTOOLCHAIN=auto` for all `go` commands (already set in `Makefile` via `export`).
- Read `docs/guidance/project.md` before implementing features or bug fixes.
- **NEVER** auto-generate this file or any documentation file. Write it from actual project knowledge.

---

## Build & Test

```bash
# Build (current platform → dist/launchpad_GOOS_GOARCH)
make local

# Lint & security
make lint           # golangci-lint run
make security-scan  # govulncheck ./...

# Unit tests — require --tags 'testing' build tag
make unit-test
go test -v --tags 'testing' ./pkg/...
go test -v --tags 'testing' ./pkg/config/... -run TestFoo   # single test

# Functional & integration
make functional-test    # test/functional/ — component level, may need network
make integration-test   # test/integration/ — requires real nodes

# Smoke tests — require AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY
make smoke-modern    # RHEL9/Ubuntu24/Rocky9, MCR stable-29.2, MKE 3.9.2  (50m)
make smoke-legacy    # RHEL8/Rocky8/Ubuntu22, MCR stable-25.0, MKE 3.8.8  (50m)
make smoke-windows   # Ubuntu24 mgr + Win2019/2022/2025, MCR stable-25.0  (60m)
make smoke-upgrade   # Install 3.8.8 → upgrade to 3.9.2, same infra        (90m)
```

---

## Architecture: Phase Manager Pattern

All operations (apply, reset, describe) are ordered sequences of **phases** run by `phase.Manager`. This is the central pattern — new features belong in a new phase or an addition to an existing one.

```
cmd/apply.go
  └── pkg/product/mke/mke.go  (Apply)
        └── phase.Manager.Run()
              └── [Phase1, Phase2, ..., PhaseN]  sequential
```

Each phase implements `Run() error` and `Title() string`. Optional: `Prepare(config)`, `ShouldRun()`, `CleanUp()`, `DisableCleanup()`.

**Key packages:**

| Package | Role |
|---|---|
| `pkg/phase/` | Phase Manager orchestration |
| `pkg/product/mke/` | Apply / reset / describe entry points |
| `pkg/product/mke/phase/` | 30+ phase implementations |
| `pkg/product/mke/config/` | Config structs: `ClusterConfig`, `Host`, `Hosts`, `MCRConfig`, `MKEConfig`, `MSRConfig` |
| `pkg/config/` | YAML parsing, schema migrations v1–v16 |
| `pkg/configurer/` | OS-specific MCR install/upgrade (EL, Ubuntu, SLES, Windows) |
| `pkg/mcr/` | MCR runtime helpers (version detect, ensure-running) |
| `pkg/swarm/` | Docker Swarm helpers (node ID, cluster ID) |
| `pkg/kubeclient/` | Kubernetes client |
| `pkg/docker/` | Image handling and auth |
| `pkg/analytics/` | Segment telemetry |

---

## Configuration Schema

Current: `apiVersion: launchpad.mirantis.com/mke/v1.6`

MCR is selected by **channel** (e.g. `stable-29.2`), not by specific version number. Migrations for older schemas live in `pkg/config/migration/` and run automatically on load.

```yaml
apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
metadata:
  name: my-cluster
spec:
  hosts:
  - role: manager
    ssh:
      address: 1.2.3.4
      user: ubuntu
      keyPath: ~/.ssh/id_rsa
  mcr:
    channel: stable-29.2
  mke:
    version: 3.9.2
    adminUsername: admin
    adminPassword: secret
```

If you change the config schema: bump `apiVersion`, add a migration in `pkg/config/migration/`, add unit tests for it.

---

## Smoke Tests

Smoke tests (`test/smoke/`) use [Terratest](https://terratest.gruntwork.io/) to provision real AWS infrastructure via `examples/terraform/aws-simple/`, run the full Launchpad lifecycle, and destroy everything unconditionally via `defer terraform.Destroy`. All resources are tagged `launchpad-smoke-test: true`.

| Make target | Test function | Timeout | What it tests |
|---|---|---|---|
| `smoke-modern` | `TestModernCluster` | 50m | Install on RHEL9/Ubuntu24/Rocky9 |
| `smoke-legacy` | `TestLegacyCluster` | 50m | Install on RHEL8/Rocky8/Ubuntu22 |
| `smoke-windows` | `TestWindowsCluster` | 60m | Install with Windows 2019/2022/2025 workers |
| `smoke-upgrade` | `TestUpgradeLegacyToModern` | 90m | Install 3.8.8 then upgrade to 3.9.2 in place |

CI jobs are gated by PR labels: `smoke-test` (all jobs), or individual labels `smoke-modern`, `smoke-legacy`, `smoke-windows`, `smoke-upgrade`.

**To add a new smoke test**, read `docs/development/smoke-tests.md` — it documents the full framework: available platforms, helper functions, how to write install/reset and upgrade tests, CI wiring, and a pre-submission checklist.

---

## Contributing

- Feature branches only — never `main`.
- Signed commits: `git commit -s`.
- New functionality → new phase, not inline logic.
- Run `make lint` and `make unit-test` before opening a PR.
- PR description must explain trade-offs and link any relevant Jira ticket (PRODENG-XXXX).

---

## Collaborative / Multi-Engineer Workflows

When multiple engineers or agents work on the same initiative:

- Communicate before modifying shared files (smoke tests, Terraform examples, CI workflow).
- Prefer additive changes (new test functions, new phases) to reduce merge conflicts.
- Use a separate file per concern in `test/smoke/` (e.g. `upgrade_test.go`) rather than growing `smoke_test.go`.
- Coordinate PR labels to avoid running expensive smoke jobs unnecessarily.
- Tag all AWS resources with `launchpad-smoke-test: true` and a descriptive `launchpad-smoke-test-name` value so each engineer's resources are identifiable in the console.

---

## Documentation

| File | Purpose |
|---|---|
| `docs/guidance/project.md` | Core architectural principles |
| `docs/specifications/architecture.md` | Phase Manager, apply/reset sequences, design decisions |
| `docs/development/workflow.md` | Build, test, and contribution workflow |
| `docs/requirements/launchpad-prd.md` | Product requirements |
| `docs/usage/getting-started.md` | User-facing getting started guide |
| `docs/development/smoke-tests.md` | How to write new smoke tests (framework, platforms, CI wiring) |

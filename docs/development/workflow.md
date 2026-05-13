# Development Workflow: Mirantis Launchpad

This guide covers building, testing, and contributing to the Launchpad codebase.

## Building Locally

```bash
make local    # Build for the current platform → dist/launchpad_GOOS_GOARCH
```

The binary version and commit hash are injected at build time via `-ldflags`. Production release builds are handled by Goreleaser in CI.

## Testing Strategy

Launchpad's system-centric nature requires a layered testing approach:

### 1. Unit Tests (`pkg/`)

Tests for individual functions and components. Require the `testing` build tag.

```bash
make unit-test
# or directly:
go test -v --tags 'testing' ./pkg/...

# Single package:
go test -v --tags 'testing' ./pkg/config/...

# Single test:
go test -v --tags 'testing' ./pkg/config/... -run TestFoo
```

### 2. Functional Tests (`test/functional/`)

Component-level tests that may require network access but do not need a live cluster.

```bash
make functional-test
```

### 3. Integration Tests (`test/integration/`)

Verify behaviour against real provisioned nodes.

```bash
make integration-test
```

### 4. Smoke Tests (`test/smoke/`)

Full end-to-end tests using [Terratest](https://terratest.gruntwork.io/). Each test provisions real AWS infrastructure via `examples/terraform/aws-simple/`, runs the complete Launchpad lifecycle (install → reset), and destroys all infrastructure unconditionally via `defer terraform.Destroy`.

Require `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.

| Make target | Test | Timeout | Description |
|---|---|---|---|
| `smoke-modern` | `TestModernCluster` | 50m | RHEL9/Ubuntu24/Rocky9, MCR stable-29.2, MKE 3.9.2 |
| `smoke-legacy` | `TestLegacyCluster` | 50m | RHEL8/Rocky8/Ubuntu22, MCR stable-25.0, MKE 3.8.8 |
| `smoke-windows` | `TestWindowsCluster` | 60m | Ubuntu24 manager + Windows 2019/2022/2025 workers |
| `smoke-upgrade` | `TestUpgradeLegacyToModern` | 90m | Install MCR stable-25.0/MKE 3.8.8, upgrade to stable-29.2/MKE 3.9.2 |

```bash
# Run a specific smoke test
make smoke-modern
make smoke-upgrade
```

All smoke-test AWS resources are tagged `launchpad-smoke-test: true` for cost tracking. CI smoke jobs are gated by PR labels (`smoke-test`, `smoke-modern`, `smoke-legacy`, `smoke-windows`, `smoke-upgrade`).

## Contributing Principles

- **Signed Commits**: All commits must be signed using `git commit -s`.
- **Feature Branches**: Never commit directly to `main`. Always work on a feature branch and open a PR.
- **Feature Options**: Make new features optional via configuration or command flags.
- **Phase Integration**: Implement new functionality as phases whenever possible for reusability.
- **Schema Safety**: Avoid changes to the configuration syntax. If a change is necessary:
  - Bump the `apiVersion` (currently `launchpad.mirantis.com/mke/v1.6`).
  - Include an in-memory migration in `pkg/config/migration/`.
  - Add unit tests for the migration.
- **Linting**: Ensure all changes pass `make lint` (`golangci-lint run`).
- **Security**: Run `make security-scan` (`govulncheck ./...`) before raising a PR.

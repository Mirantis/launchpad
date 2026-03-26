# Development Workflow: Mirantis Launchpad

This guide covers building, testing, and contributing to the Launchpad codebase.

## Building Locally

Launchpad uses `goreleaser` for production builds, but provides a `Makefile` for local development.

- **`make local`**: Builds a single, platform-specific binary for rapid testing.
- **`make build-release`**: Performs a full production build, requiring a clean repository and release tag.
- **`make sign-release`**: Signs the Windows binary (requires specific environment variables).

## Testing Strategy

Launchpad's system-centric nature requires a layered testing approach:

### 1. Unit and Functional Tests (`pkg/`, `test/functional/`)
- **Unit**: Small tests for individual functions or components.
- **Functional**: Tests that verify specific functional components.
- **Run**: `go test ./pkg/...` and `go test ./test/functional/...`.

### 2. Integration Tests (`test/integration/`)
- **Focus**: Verifies functional elements on actual clusters provisioned by the test suite.
- **Run**: `go test ./test/integration/...`.

### 3. Smoke Tests (`test/smoke/`)
- **Focus**: End-to-end command testing (`apply`, `reset`, etc.) using a real compute cluster.
- **Smoke Small**: Provision a small number of machines.
- **Smoke Large**: Provision a large and varied cluster.
- **Run**: `go test ./test/smoke/...`.

## Contributing Principles

- **Signed Commits**: All commits must be signed using `git commit -s`.
- **Feature Options**: Make new features optional via configuration or command flags.
- **Phase Integration**: Implement new functionality as phases whenever possible for reusability.
- **Schema Safety**: Avoid changes to the configuration syntax. If a change is necessary:
  - Bump the version.
  - Include an in-memory migration in `pkg/config/migration/`.
- **Linting**: Ensure all changes pass `golangci-lint`.

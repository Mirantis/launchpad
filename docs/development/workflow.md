# Development Workflow: Mirantis Launchpad

This guide covers building, testing, and contributing to the Launchpad codebase.

## Building Locally

Launchpad provides a `Makefile` for local development. Production release builds run in GitHub Actions (`.github/workflows/release.yml`).

- **`make local`**: Builds a single, platform-specific binary for rapid testing.

## Cutting a Release

A release is produced by pushing a tag. The tag's shape, and nothing else,
decides what the workflow creates:

| Tag | Example | Result |
| --- | --- | --- |
| `vX.Y.Z` | `v1.5.20` | Draft, full release |
| `vX.Y.Z-rcN` or `-tpN` | `v1.5.20-rc1`, `v1.5.20-tp2` | Draft, prerelease |
| `vX.Y.Z-alphaN` or `-betaN` | `v1.5.20-alpha1` | Draft, prerelease |
| Anything else | `v1.5.20-PRODENG-1234` | **No release.** Binaries are uploaded as a workflow artifact only |

The classification lives in `.github/scripts/release-state.sh` and is printed
in the run summary. Run it locally to check a tag before pushing it:

```bash
./.github/scripts/release-state.sh v1.5.20-rc1
```

**Every release is created as a draft, and a draft is not visible to anyone
without push access.** The workflow's job ends there. A maintainer then reviews
the assembled release notes (see `.changes/README.md`) and the attached
binaries, and publishes the release from the GitHub UI. Whether the published
release becomes "Latest" is chosen at that point — GitHub cannot mark a draft
as latest, so the workflow does not set it.

Use a non-release tag shape when you want built binaries without a release:
the build job runs in full and attaches them to the workflow run. This is the
supported way to get a one-off build for a ticket, and it is why such tags
deliberately produce nothing a maintainer has to delete afterwards.

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

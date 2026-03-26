# Mirantis Corporate Guidance for Launchpad

Launchpad is an open-source tool steered by Mirantis to provide a seamless installation experience for MKE and MCR clusters.

## Steering Principles

- **Avoid Breaking Changes**: Backward compatibility is a primary concern. Any changes to the configuration schema must include a migration strategy.
- **Maintain Focus**: Avoid adding complexity that serves only niche or rare use-cases.
- **Consistency**: The tool must align with Mirantis product standards and existing ecosystem tools like `testkit`.

## Contribution Rules (Signed Commits Required)

- **Issues**: All changes should have a corresponding GitHub issue for high-level discussion before implementation.
- **Pull Requests**:
  - Maintain a 1-to-1 relationship between PRs and issues whenever possible.
  - Commits MUST be signed (use `git commit -s`).
  - PRs must be rebased off of the `main` branch.
  - Changes must pass all `golangci-lint` checks.
- **Contact**: Ping `@james-nesbitt` on GitHub for feedback if it is not received in a timely manner.

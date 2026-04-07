# AI Agent Guidelines for Mirantis Launchpad

This document defines guidelines for AI agents working on the Launchpad project. These instructions take precedence over general tool defaults.

---

## Core Principles

1. **Generalization**: Support all AI agents, not just specific providers.
2. **Consistency**: Follow Launchpad’s architectural patterns and conventions.
3. **Clarity**: Document decisions and reasoning for human reviewers.
4. **Branching**: NEVER work on, push to, or merge to the `main` branch. All work MUST be done on feature branches.

---

## Documentation Structure

Documentation is organized to minimize context overhead for agents:
- `docs/guidance/`: Foundational principles and project vision.
- `docs/requirements/`: High-level product requirements (PRDs).
- `docs/specifications/`: Technical specifications, architecture, and design.
- `docs/development/`: Development workflows, building, and testing.
- `docs/usage/`: User-facing documentation.

---

## Agent Workflow

### 1. Research Phase
- Read `docs/guidance/project.md` to understand core architectural principles.
- For feature requests, review the relevant PRD in `docs/requirements/`.
- For bug fixes, review the relevant specification in `docs/specifications/`.
- **Environment**: Ensure `GOTOOLCHAIN=auto` is used for all `go` commands to support the required toolchain.

### 2. Strategy Phase
- **ALWAYS** create a new feature branch from `main`.
- Propose changes that align with the **Phase Manager** architecture (`docs/specifications/architecture.md`).
- Ensure changes are backwards compatible or include migrations (`docs/specifications/architecture.md`).
- Document trade-offs and decisions in the PR description.

### 3. Execution Phase
- **NEVER** push directly to `main`.
- **NEVER** merge into `main`.
- All work MUST be pushed to a remote feature branch for human review.
- All commits MUST be signed.
- Follow the testing strategy outlined in `docs/development/workflow.md`.
- Use `make local` for rapid iteration and validation.

---

## Technical Constraints

- **Language**: Go (Golang).
- **Core Library**: [k0sproject Rig](https://github.com/k0sproject/rig) for host management.
- **Build System**: Goreleaser (invoked via `Makefile`).
- **Telemetry**: Maintain existing telemetry patterns for installation, upgrades, and errors.
- **State**: Launchpad is stateless between runs; use phases for discovery.

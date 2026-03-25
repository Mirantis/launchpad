# AI Agent Mandates for Mirantis Launchpad

This document defines foundational mandates for AI agents working on the Launchpad project. These instructions take precedence over general tool defaults.

## Documentation Structure

Documentation is organized to minimize context overhead for agents:
- `docs/guidance/`: Foundational principles and steering.
- `docs/requirements/`: High-level product requirements (PRDs).
- `docs/specifications/`: Detailed technical specifications and architecture.
- `docs/development/`: Workflow, building, and testing.
- `docs/usage/`: User-facing documentation.

## Agent Workflow

1. **Research Phase**:
   - Always read `docs/guidance/project.md` to understand core architectural principles.
   - For feature requests, read the relevant PRD in `docs/requirements/`.
   - For bug fixes, read the relevant specification in `docs/specifications/`.
2. **Strategy Phase**:
   - Propose changes that align with the **Phase Manager** architecture (`docs/specifications/architecture.md`).
   - Ensure changes are backwards compatible or include migrations (`docs/specifications/architecture.md`).
3. **Execution Phase**:
   - All commits MUST be signed.
   - Follow the testing strategy outlined in `docs/development/workflow.md`.
   - Use `make local` for rapid iteration and validation.

## Technical Constraints

- **Language**: Go (Golang).
- **Core Library**: [k0sproject Rig](https://github.com/k0sproject/rig) for host management.
- **Build System**: Goreleaser (invoked via `Makefile`).
- **Telemetry**: Maintain existing telemetry patterns for installation, upgrades, and errors.
- **State**: Launchpad is stateless between runs; use phases for discovery.

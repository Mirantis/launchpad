# Launchpad Project Guidance

Launchpad is a Go-based CLI tool designed for installing, upgrading, and resetting MKE and MCR clusters.

## Core Architectural Principles

- **Infrastructure Agnostic**: The tool must work on any infrastructure (on-prem, public cloud, private cloud, hybrid, bare-metal).
- **No Mandatory External Dependencies**: Do not require pre-installed tools on cluster machines.
- **Built-in Telemetry**: Always include telemetry for installation and upgrade phases, including error reporting.
- **Diagnostic-Focused**: Provide meaningful output for diagnostics (e.g., `error.log`, `install.log`).
- **Statelessness**: Launchpad does not maintain persistent state between runs. Use discovery phases to determine cluster state at runtime.

## Technical Framework

- **Go (Golang)**: The primary language for the CLI and its core components.
- **k0sproject Rig**: Used for managing compute nodes/machines over SSH or WinRM.
- **Phase Manager**: All operations (install, upgrade, reset) are organized into reusable **Phases** that can be conditionally executed.
- **Configuration Migrations**: Maintain backward compatibility by providing in-memory migrations for older configuration formats.

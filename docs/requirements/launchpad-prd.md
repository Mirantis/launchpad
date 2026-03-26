# Product Requirements Document (PRD): Mirantis Launchpad

Launchpad is a single-binary CLI tool to assist with installing, upgrading, and resetting MKE and MCR clusters on provisioned compute node resources.

## Objectives

- **Automate Installation**: Streamline the setup of MKE and MCR on a set of compute nodes.
- **Support Upgrades**: Provide a seamless path for upgrading cluster components.
- **Infrastructure Agnostic**: Allow users to provision their own infrastructure using any technology (e.g., Terraform, CloudFormation, manual setup).
- **Simplicity**: Be easy to use for both local development and production deployments.

## Core Features

- **Static YAML Configuration**: A declarative file defining hosts and product details.
- **Cross-Platform Support**: Binaries for Linux, macOS, and Windows.
- **Node Management**: Support for both Linux and Windows compute nodes via SSH and WinRM.
- **Telemetry and Diagnostics**: Detailed logging and telemetry for all major operations.

## Target Audience

- **Cloud Architects**: Designing Mirantis environments.
- **System Administrators**: Managing MKE/MCR clusters.
- **DevOps Engineers**: Automating infrastructure workflows.
- **Developers**: Testing cluster features locally or in a dev environment.

## Design Goals

- **Performance**: Rapid execution of installation phases.
- **Reliability**: Resilient to networking issues or temporary node failures.
- **Ease of Integration**: Easily used within CI/CD pipelines.

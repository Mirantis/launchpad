# Mirantis Launchpad CLI Tool

The Mirantis Launchpad CLI tool automates the installation, upgrading, and resetting of MKE (Mirantis Kubernetes Engine) and MCR (Mirantis Container Runtime) clusters on provisioned compute node resources.

## Documentation Index

Documentation is organized to provide clear, context-efficient guidance for both users and AI agents:

- **[AI Agent Mandates](GEMINI.md)**: Foundational instructions for AI agents working in this repository.
- **[Requirements (PRDs)](docs/requirements/launchpad-prd.md)**: High-level objectives and product requirements.
- **[Corporate Guidance](docs/guidance/corporate.md)**: Steering and contribution principles from Mirantis.
- **[Project Guidance](docs/guidance/project.md)**: Core architectural principles and technical frameworks.
- **[Technical Specification](docs/specifications/architecture.md)**: Details on the Phase Manager and internal components.
- **[Development Workflow](docs/development/workflow.md)**: Building, testing, and contribution rules.
- **[User Guide](docs/usage/getting-started.md)**: Installation, provisioning, and running instructions.

## Quick Start

1.  **Download**: Get the latest binary from [GitHub Releases](https://github.com/Mirantis/launchpad/releases).
2.  **Provision**: Set up your compute nodes (optionally using [Mirantis Terraform helpers](docs/usage/getting-started.md#terraform-helpers)).
3.  **Configure**: Create a `launchpad.yaml`.
4.  **Execute**: Run `launchpad apply`.

For detailed usage, refer to the [public documentation](https://docs.mirantis.com/mke/3.8/launchpad.html).

## Contributing

We welcome contributions! Please read our [Contribution Rules](docs/guidance/corporate.md#contribution-rules-signed-commits-required) and [Development Workflow](docs/development/workflow.md) before submitting a pull request.

**All commits MUST be signed.**

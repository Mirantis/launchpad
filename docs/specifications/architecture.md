# Technical Architecture Specification: Launchpad

This document outlines the internal architecture of Launchpad, emphasizing its stateless, phase-based execution model.

## Component Model

### Configuration Management (`pkg/config/`)

- **YAML-driven**: Launchpad interprets a static configuration file (`launchpad.yaml` by default).
- **Structure**:
  - `hosts`: A list of compute nodes and their roles.
  - `mke`: A configuration block specific to the Mirantis Kubernetes Engine (MKE) product.
- **Migrations**: Found in `pkg/config/migration/`, these transform older versions of the config into the current internal representation at runtime.

### Host Management (`k0sproject Rig`)

- **Role**: Rig manages the low-level connection to compute nodes (SSH for Linux, WinRM for Windows).
- **Functionality**: Executing remote commands, uploading files, and managing node-level state.
- **Integration**: Launchpad passes host definitions directly to Rig.

### Phase Manager (`pkg/phase/`)

- **Concept**: All actions are organized into a sequence of **Phases**.
- **Execution**: The manager runs each phase in order, stopping if an error is encountered.
- **Reusability**: Phases are modular and can be reused across different commands (e.g., `apply` and `reset`).
- **Phase Logic**: Phases should ideally detect if they need to run rather than relying on external flags.

### Product Support (`pkg/product/`)

- **`mke`**: The main supported product, covering the MCR (Mirantis Container Runtime), MKE, and MSR (Mirantis Secure Registry) stack.
- **Structs**: Product configurations and state structs are defined in `pkg/product/mke/api`.

## Command Execution Flow

1. **Load Configuration**: Read and migrate the YAML file.
2. **Initialize Phases**: Instantiate the required phases for the requested command.
3. **Execute Phases**: The Phase Manager runs the sequence, communicating with hosts via Rig.
4. **Finalize**: Generate logs and diagnostic reports.

## Persistence and State

- **Statelessness**: No persistent state is kept between runs.
- **Discovery**: Phases are responsible for identifying the current state of the cluster by querying the nodes directly.

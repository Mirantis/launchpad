# Technical Architecture Specification: Launchpad

This document outlines the internal architecture of Launchpad, emphasising its stateless, phase-based execution model.

## Component Model

### Configuration Management (`pkg/config/`)

- **YAML-driven**: Launchpad interprets a static configuration file (`launchpad.yaml` by default).
- **Current schema**: `apiVersion: launchpad.mirantis.com/mke/v1.6`
- **Structure**:
  - `spec.hosts`: A list of compute nodes and their roles (manager, worker, msr).
  - `spec.mcr`: MCR configuration — channel-based (e.g. `stable-29.2`), repo URL, Windows installer URL.
  - `spec.mke`: MKE configuration — version, image repo, admin credentials, install/upgrade flags.
  - `spec.msr`: Optional MSR configuration — version, replica IDs, TLS.
- **Migrations**: Found in `pkg/config/migration/`, these transform older versions of the config into the current internal representation at runtime. Each migration is independently unit-tested.

### Host Management (`k0sproject Rig`)

- **Role**: Rig manages the low-level connection to compute nodes (SSH for Linux, WinRM for Windows).
- **Functionality**: Executing remote commands, uploading files, and managing node-level state.
- **Integration**: Launchpad passes host definitions directly to Rig; phases receive the connected host set.

### Phase Manager (`pkg/phase/`)

- **Concept**: All actions are organised into a sequence of **Phases**.
- **Execution**: The manager runs each phase in order, stopping if an error is encountered (unless cleanup is disabled).
- **Reusability**: Phases are modular and can be reused across different commands (e.g. `apply` and `reset`).
- **Phase logic**: Phases should detect whether they need to run via `ShouldRun()` rather than relying on external flags.
- **Phase interface**:
  - `Run() error` — required
  - `Title() string` — required
  - Optional: `Prepare(config)`, `ShouldRun()`, `CleanUp()`, `DisableCleanup()`

### Product Support (`pkg/product/`)

- **`mke`**: The main supported product, covering MCR (Mirantis Container Runtime), MKE, and MSR (Mirantis Secure Registry).
- **Config structs**: Defined in `pkg/product/mke/config/` — `ClusterConfig`, `ClusterSpec`, `Host`, `Hosts`, `MCRConfig`, `MKEConfig`, `MSRConfig`.
- **OS configurers**: Distro-specific MCR install/upgrade logic lives in `pkg/configurer/` (EL, Ubuntu, SLES, Windows). Each configurer implements `InstallMCR`, `UpgradeMCR`, and related host-setup methods using the native package manager (yum/apt/zypper; PowerShell for Windows).

## Command Execution Flow

1. **Load Configuration**: Read and migrate the YAML file to the current schema version.
2. **Initialise Phases**: Instantiate the required phases for the requested command (apply, reset, describe).
3. **Execute Phases**: The Phase Manager runs the sequence, communicating with hosts via Rig.
4. **Finalise**: Emit logs, diagnostic output, and telemetry events.

## Apply Phase Sequence (abridged)

```
UpgradeCheck → Connect → DetectOS → GatherFacts → ValidateFacts → PrepareHost
→ ConfigureMCR → InstallMCR → UpgradeMCR → InstallMCRLicense → RestartMCR
→ PullMKEImages → InitSwarm → InstallMKECerts → InstallMKE → UpgradeMKE
→ JoinManagers → JoinWorkers
→ InstallMSR → UpgradeMSR → JoinMSRReplicas
→ LabelNodes → RemoveNodes → Disconnect → Info
```

`InstallMCR` and `UpgradeMCR` are separate phases; `UpgradeMCR` skips hosts where MCR was just installed. `InstallMKE` and `UpgradeMKE` are likewise separate — `UpgradeMKE` is a no-op if the installed version matches the target.

## Reset Phase Sequence

```
Connect → DetectOS → GatherFacts → PrepareHost
→ UninstallMSR → UninstallMKE → UninstallMCR → CleanUp → Disconnect
```

`UninstallMKE` runs the `mirantis/ucp uninstall-ucp` bootstrapper. If that times out (a known failure mode on large or mixed-OS clusters where agent image pulls exhaust the hardcoded deadline), it falls back to a forced swarm dissolution: removing the stuck `ucp-uninstall-agent` service, then forcing all nodes to leave the swarm sequentially.

## Persistence and State

- **Statelessness**: No persistent state is kept between runs.
- **Discovery**: The `GatherFacts` phase queries each node to determine installed MCR version, MKE installed state, swarm membership, and node ID. Subsequent phases use this metadata to decide whether to install, upgrade, or skip.

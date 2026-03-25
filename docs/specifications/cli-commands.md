# CLI Command Specification: Launchpad

Launchpad provides several core commands, each implemented as an entry point in `cmd/`.

## Primary Commands

### `apply` (`cmd/apply.go`)

- **Description**: Installs and upgrades Mirantis products (MKE, MCR, MSR) onto the hosts defined in the configuration.
- **Workflow**:
  - Load and migrate the configuration file.
  - Run the `apply` sequence of phases.
- **Key Options**:
  - `--config`: Specify the path to the configuration file.

### `reset` (`cmd/reset.go`)

- **Description**: Removes all Mirantis products from the hosts defined in the configuration.
- **Important**: This command does NOT return hosts to their pre-install state but removes the managed products.

### `exec` (`cmd/exec.go`)

- **Description**: Executes a command or opens a shell on a set of hosts defined in the configuration.
- **Usage**: Useful for running manual troubleshooting commands across the cluster.

### `help`

- **Description**: Provides detailed usage information for any command or sub-command.

## Support and Configuration Flags

Most commands support common flags:
- `--config`: Custom path to `launchpad.yaml`.
- `--debug`: Enable verbose logging for troubleshooting.
- `--log-file`: Path to store installation logs.

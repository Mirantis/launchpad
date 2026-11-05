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

### `describe` (`cmd/describe.go`)

- **Description**: Reports cluster state: `hosts`, `mke`, `msr`, or `config`.
- **`describe config`**: Prints the fully migrated and defaulted cluster configuration. Credential-bearing values are replaced with `[REDACTED]`: the MKE admin username and password, host connection passwords, inline certificate and key data, and the arguments of credential-carrying install/upgrade flags such as `--admin-password` and `--ucp-password`. Pass `--disable-redact` to print them verbatim.

### `help`

- **Description**: Provides detailed usage information for any command or sub-command.

## Support and Configuration Flags

Most commands support common flags:
- `--config`: Custom path to `launchpad.yaml`.
- `--debug`: Enable verbose logging for troubleshooting.
- `--log-file`: Path to store installation logs.

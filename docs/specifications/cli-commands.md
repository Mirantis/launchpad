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
- **`describe config`**: Prints the fully migrated and defaulted cluster configuration, with credential-bearing values replaced by `[REDACTED]`. Masked: the MKE admin username and password; host connection passwords; inline certificate and key data (`certData`, `keyData`); the MCR license content; `configData`, which holds whatever `spec.mke.configFile` and `spec.mke.cloud.configFile` contained; host `environment` variables whose name contains `PASSWORD`, `PASSWD`, `SECRET`, `TOKEN`, `CREDENTIAL`, `APIKEY` or `_KEY`; and the arguments of credential-carrying install and upgrade flags such as `--admin-password` and `--ucp-password`.
- **Not masked**: `caCertData`, which is public trust material; operational environment variables such as `HTTP_PROXY`; and shell command fields (`hooks`, `swarmUpdateCommands`), where a credential sits behind a single-letter flag whose meaning depends on the command — `-p` is a password to `docker login` and a port publication to `docker run` — so a credential placed there is printed. Masking `configData` whole costs the report its MKE `config.toml` contents; `--disable-redact` prints everything verbatim.

### `help`

- **Description**: Provides detailed usage information for any command or sub-command.

## Support and Configuration Flags

Most commands support common flags:
- `--config`: Custom path to `launchpad.yaml`.
- `--debug`: Enable verbose logging for troubleshooting.
- `--log-file`: Path to store installation logs.

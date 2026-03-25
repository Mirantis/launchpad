# Getting Started with Mirantis Launchpad

Launchpad is a single-binary CLI for managing MKE and MCR clusters.

## Installation

Download the binary for your platform (Linux, macOS, or Windows) from the [latest GitHub releases](https://github.com/Mirantis/launchpad/releases).

For Linux and macOS (AMD64 and ARM64):
- Download the binary.
- Make it executable: `chmod +x launchpad`.
- Move it to your path: `sudo mv launchpad /usr/local/bin/`.

## Provisioning Compute Nodes

Before using Launchpad, you must provision your compute nodes. Ensure the nodes follow the [Mirantis predeployment documentation](https://docs.mirantis.com/mke/3.8/install/predeployment.html).

### Terraform Helpers
Mirantis provides Terraform modules for various clouds:
- **AWS**: [provision-aws](https://github.com/terraform-mirantis-modules/terraform-mirantis-provision-aws)
- **Azure**: [provision-azure](https://github.com/terraform-mirantis-modules/terraform-mirantis-provision-azure)
- **GCP**: [provision-gcp](https://github.com/terraform-mirantis-modules/terraform-mirantis-provision-gcp)

## Configuration (`launchpad.yaml`)

Create a `launchpad.yaml` to define your cluster. A minimal configuration includes:
- **Hosts**: IP addresses, usernames, and roles (manager, worker).
- **MKE Product**: Specific versions and license information.

For a full reference, see the [public documentation](https://docs.mirantis.com/mke/3.8/launchpad.html).

## Running Launchpad

Use the `apply` command to start the installation:
```bash
launchpad apply --config launchpad.yaml
```

To remove managed products from the cluster:
```bash
launchpad reset --config launchpad.yaml
```

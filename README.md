# Mirantis Launchpad CLI Tool

The Mirantis custom binary cli tool to assist installing, upgrading and resetting MKE3 & MCR clusters on provisioned compute node resources. Launchpad provides advantages over manually installing MCR and MKE3 in it automates the process without limiting how the underlying cluster is provisioned or managed. The tool is a golang cli written leveraging the k0sproject rig library for managing compute nodes/machines.

Launchpad is an open source tool, steered by Mirantis.

Launchpad was open sourced in early 2025, during which time development moved from an internal process to a GitHub based bublic process. Older pull requests include internal ticket numbers for reference.

## Development 

Issues are reported as Github issues. Feature requests can also be submitted as Issues.

Pull requests are accepted, but it is requested that contributors first read through the [developer's guide](docs/developer.md)

Steering is heavily guided by Mirantis, mainly in order to avoid breaking changes, and to avoid adding complexity that serves only rare use-cases.

## Usage 

Launchpad is installed pulling directly from the Launchpad GitHub releases. Before executing, clusters need to be provisioned and a configuration file needs to be provided.

Users can get oriented using the [usage guide](docs/usage.md) but should read the [public documentation](https://docs.mirantis.com/mke/3.8/launchpad.html) for real usage.

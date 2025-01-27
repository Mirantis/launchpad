# Launchpad usage

Consider reading the full public [Launchpad documentation](https://docs.mirantis.com/mke/3.8/launchpad.html). Here you can find a short description of how launchpad can be used.

## Installing the launchpad tool

The launchpad binary can be downloaded directly from the latest GitHub release. AMD64 and ARM64 releases are available for linux, osx, windows and more.
If your preferred platform isn't available, launchpad is still easy to build, simbly follow instruction in the [developer guide](developer.md)

Once the binary is downloaded it can be used by directly using the binary.

## Provisioning

Launchpad requires a compute node cluster to be available for installation, before the tool can be used. This cluster is typically a set of virtual machines, networked togthers.
Because provisioning is such an opinionated but diverse topic, launchpad doesn't provision, but rather allows you to provision using whatever process you want, as long as the cluster is configured correctly accordint to the [Mirantis documentation](https://docs.mirantis.com/mke/3.8/install/predeployment.html), and is accessible to launchpad.

### Mirantis terraform helpers 

Mirantis provides a number of cloud specific terraform modules that can be used to provision clusters ready to be used with launchpad:

#### Launchpad provision

A set of modules that can provision infrastucture for both launchpad and k0sctl, with examples showing how to use them with both the launchpad binary (see the local variables used to create launchpad yaml) and the launchpad terraform provider.

1. [AWS](https://github.com/terraform-mirantis-modules/terraform-mirantis-provision-aws/tree/main): https://registry.terraform.io/modules/terraform-mirantis-modules/provision-aws/mirantis/latest
2. [Azure](https://github.com/terraform-mirantis-modules/terraform-mirantis-provision-azure/tree/main): https://registry.terraform.io/modules/terraform-mirantis-modules/provision-azure/mirantis/latest
2. [GCP](https://github.com/terraform-mirantis-modules/terraform-mirantis-provision-gcp/tree/main): https://registry.terraform.io/modules/terraform-mirantis-modules/provision-gcp/mirantis/latest

Source code is available, and the modules can be directly used in Terraform.

## Configure

In order to run most commands, launchpad requires a configuration file, describing the cluster hosts, and the product details. The configuration is described well in the [public documentation](https://docs.mirantis.com/mke/3.8/launchpad.html)

By default, the file is a yaml file called `launchpad.yaml`.

## Running 

Run launchpad by executing the binary with a command argument and related flags. Commands can be discovered using the `help` command. If no configuration file flag is set, then launchpad should be executed in the path containing the configuration file.

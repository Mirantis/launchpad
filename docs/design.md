# Launchpad design guide

We will draw inspiration from existing tooling such as [Docker Cluster](https://github.com/Mirantis/cluster) and [testkit](https://github.com/Mirantis/testkit)

## Design goals

* Infrastructure agnostic (works on any infra; on-prem, public cloud, private cloud, hybrid, baremetal)
* No mandatory dependencies to other tools or prerequisites to install anything beforehand to cluster machines
* Built-in telemetry (Installation & Upgrade related; errors too)
* Will provide meaningful output for diagnostics (e.g. error.log / install.log)
* Support common Operating systems
* Support for infrastructure management any provisioning technology, extensive Terraform support available
* The tool is convenient to use
* The tool is flexible where possible, but launchpad it not mean solve all possible problems.

## Design

### Config 

Launchpad operates primarily by interpreting a static yaml configuration file.

The file matches a know syntax, primarily defining a list of compute node `hosts`, and a per product configuration block. The syntax guidelines for the configuration file are best tracked on the public documentation or in the [short usage guide](usage.md)

Launchpad maintains backward support for config file changes, by including migrations to transform syntax in memory at runtime.

The Launchpad product configurations are directly unmarshalled into structs defined on the product.

### Hosts

Mirantis MCR and MKE are designed to run on a set of compute hosts running Linux or Windows operating systems. Each compute node is referred to as a host.
Launchpad configuration includes a mandatory section, where Hosts are defined. A host definition includes connection parameters, along with a role definition for how MCR will use the node in the cluster.

In launchpad, hosts are considered to be connectable machines, using either SSH or WinRM. [k0sproject Rig](https://github.com/k0sproject/rig) is used to interface with the machines, and launchpad configuration for a host is directly passed to Rig, making it easy to define how to connect.

Note that launchpad considers only the hosts that are described in the configuration file. Hosts that are in the cluster but not in the configuration file are not managed by launchpad.

### Commands 

Commands are ingresses into the launchpad execution, which declare what actions are going to be taken.

You can find the command code in the `/cmd` folder. Commands are generally put into their own file, with a generic support for common flags which load the config.

Some of the more important commands:

1. [apply](../cmd/apply.go): install Mirantis products onto the config defined nodes
2. [reset](../cmd/reset.go): remove all of the Mirantis products from the config defined hosts * the command is not designed to return hosts to pre-install state *
3. [exec](../cmd/exec.go): execute commands, or get a shell on a set of hosts.

### Products

Launchpad was originally designed to be product agnostic, with a hope that the tool could be adapted for various installation profiles. In the end, this abstraction turned only into a thin command layer abstraction and more of a hindrance than a value. 

Launchpad has only one product: `mke`. The single product refers to an MCR & MKE stack, typically as a kubernetes distribution. The MKE product can also install MSR. The launchpad MKE versions installable are limited to the 3.7 and 3.8 versions (not MKE4.) The MSR versions installable are limited to the legacy 2.9 versions.

The MKE product config and state structs can be found in the `pkg/product/mke/api`. The MCR state are spread across hosts and the `pkg/product/common/api` files.

#### Product state

Launchpad maintains no state between runs. Launchpad products are expected to include phases which discover the state of the cluster. Typically the state metadata is kept in internal data structures that are objects on the configuration structs.

### Phases

Launchpad typically executes commands using a sequence of phases. Each phase executes some functionality and optionally modifies the various state structs. Phases can be made optional deciding at run time if they should run.

In order to manage the phases, launchpad includes a phase manager, to which phases are added. The manager takes responsibility for executing the phases, and interupting flow is a phase encounters an error.

The source code for the phase manager can be found in `pkg/phase`

# mcc - a.k.a Mirantis Cluster Control


MCC is a tool to control the lifecycle of UCP clusters.


## Design goals

Based on brainstorming session, we decided to spike a following tool:

* Goal: "from zero to hero" in 5mins; no need to read docs; install new clusters from scratch or upgrade existing
* Infrastructure agnostic (works on any infra; on-prem, public cloud, private cloud, hybrid, baremetal)
    * But should suport some sort of "integration" flow with infra mgmt tooling (e.g. Terraform)
* No mandatory dependencies to other tools or prerequisites to install anything beforehand to cluster machines
* Spike target: Support Ubuntu 18.04 & RHEL 8 host OS

We will draw inspiration from existing tooling such as [Docker Cluster](https://github.com/Mirantis/cluster) and [testkit](https://github.com/Mirantis/testkit)


## Development

As this is a pure spike we aim for proving the functionality and requirements can be fulfilled rather than for absolute code quality and test coverage.
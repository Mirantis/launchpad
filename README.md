# Mirantis Cluster Control ("mcc") CLI Tool

> The Next Generation UCP Installer & Lifecycle Management Tool

The purpose of `mmc` is to provide amazing new user experience for anyone interested in getting started with UCP product. It will simplify the complex installation process and provides "from zero to hero" experience in less than 5mins for IT admin / DevOps people who are experienced with various command line tools and cloud technologies. In addition, it'll provide functionality to upgrade existing UCP clusters to new versions with no downtime or service interruptions (high availability clusters). In the future, more functionality may be added.

## Design goals

* Infrastructure agnostic (works on any infra; on-prem, public cloud, private cloud, hybrid, baremetal)
* No mandatory dependencies to other tools or prerequisites to install anything beforehand to cluster machines
* Ultra Fast (as fast as possible given the stack we are dealing with)
* Multi cluster management (see all clusters, their health, running versions)
* Built-in telemetry (Installation & Upgrade related; errors too)
* Will provide meaningful output for diagnostics (e.g. error.log / install.log)
* Support Ubuntu 18.04 & RHEL 7/8 & CentOS 7/8 (+easily add new host OS support)
* Support for infrastructure management via Terraform
* Bonus: Possibility to use as an API/library for 3rd party tools & services & products such as MCM

We will draw inspiration from existing tooling such as [Docker Cluster](https://github.com/Mirantis/cluster) and [testkit](https://github.com/Mirantis/testkit)

## Development

Based on brainstorming session, we decided to spike the tool with an idea to prove it can be done and to gather some more ideas from the field & sales organizations after there is something to show. Very soon after having the first prototypes working, the scope was changed to make it available for Barracuda release.

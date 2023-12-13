# Mirantis Launchpad CLI Tool

> The Next Generation Mirantis Cluster Installer & Lifecycle Management Tool

The purpose of `launchpad` is to provide an amazing new user experience for anyone interested in getting started with cluster products. It will simplify the complex installation processes and provides "from zero to hero" experience in less than 5mins for IT admin / DevOps people who are experienced with various command line tools and cloud technologies. In addition, it'll provide functionality to upgrade existing clusters to new versions with no downtime or service interruptions (high availability clusters). In the future, more functionality may be added.

See the public Github repo for getting started instructions, documentation and more at https://github.com/mirantis/launchpad.

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

## Test Plan

* **PR Tests** - There are CI tests that are run on Jenkins for each pull request. Today, the tests will cover basic testing such as linter, unit tests and elementary smoke tests (simple integration tests).
* **Integration Tests** - At this stage only elementary smoke tests are included to test the product on various host OS environments such as CentOS7/8 and Ubuntu 18.04.
* **Manual Tests** - We hope QA team would run some of their existing test plans manually on clusters created with `launchpad`.

## Building

To build `launchpad` run:

```
make build
```

If you receive permission denied errors when building the builder image, ensure you have an `ssh-agent` running and have added the private key you use with GitHub to the `ssh-agent`, you can verify the agent is running and a key has been added with:

```
ssh-add -l
```

If no keys are present or you do not recognize your GitHub private key within the output above add it with:

```
ssh-add path/to/private/key
```

## Release Process

Releases are made from git tags by CICD system. The release builds must be triggered manually. The release process is the following:

1. Create new or update the existing release branch
2. Create new tag, for example `git tag 1.1.0 && git push --tags origin`
3. Go to [Jenkins](https://ci.docker.com/teams-orchestration/job/mcc/job/mcc/view/tags/) and select `Build now` from the dropdown menu of the corresponding tag to trigger the release build.
4. After the release build is ready, go to [Launchpad releases](https://github.com/Mirantis/launchpad/releases) in GitHub. Edit the draft release, write the changelog in the description field and publish the release.

## Comparison to Existing Tools

Overall, none of the pre-existing tools provide the convenience and flexibility we'd like to have for a tool that is designed for self service with an amazing new user experience.

* **Convenience** means user experience; minimizing prerequisites, must read docs, domain specific knowledge and actual steps required for performing the install (or upgrade).
* **Flexibility** means this tool should work in any environment, system or infrastructure the user might have already available for the purpose (=evaluation, running in prod or other). Note: with _any environment_ we should mean literally any. AWS, Azure, GCP, VMware, RH satellite managed systems, OpenStack, Mirantis private cloud where machines are ordered via IT tickets, any other private datacenter or whatever public cloud provider, rack of machines in your own home closet... etc.

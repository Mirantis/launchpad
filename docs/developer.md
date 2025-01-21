# Launchpad developer guide

## Building and working locally 

Launchpad is build using [goreleaser](https://goreleaser.com), but does not depend on any enterprise features. There are [make](https://www.gnu.org/software/make/) targets to simplify the build and ensure that required variables are initialized.

There are two targets to be aware of:

1. `make build-release` builds a full production build, requiring a clean repo and release tag. This is typically combined with `make sign-release` which ensures that the windows binary is signed.
2. `make local` builds a single, local platform, simple snapshot build for local testing.

## Test Plan

1. Unit tests - unit tests exist where they can, but the heavy system orientation of launchpad leaves the majority of the code base unstestable.
2. Functional tests - tests to test functional components
3. Integration tests - testing of particular functional elements that are run on actual clusters, provisioned by the test suite.
4. Smoke tests - basic end-to-end command testing for launchpad using an actual compute cluster. Currently tied to terraform for cluster provisioning. There are two smoke tests, as small and a large, differing in the number and variety of machines provisioned.

Unit and functional tests work using go test as is common for all golang projects. Functional tests are collected in `test/functional`.

Smoke and integration tests can be run by the golang testing framework in `test/integration` and `test/smoke`. The testing framework will first provision clusters and then run the tests; clusters are torn down after the test is completed.

## Issues 

> Launchpad moved to github issues in 2025 January, as a part of moving to open source management of the tooling

github.com/Mirantis/launchpad/issues

If you discover a bug, or want to request a feature for launchpad, feel free to open an issue on the Github project.

## Pull Requests (Contributing)

Mirantis welcomes contributions to the launchpad codebase, however we do heavily steer the project. This means that the acceptance of code changes requires the contribution meet the following criteria:

1. The change follows the [design](design.md) goals of the product
2. The change adds meaningful value to the tool
3. The change considers the impact on all systems/clusters/platforms, and not just the system used by the contributor
4. Pull requests deliver issues (1-1 reletionship between PRs and issues is preferred; ask for an exception if it makes sense)
5. Pull requests should include only signed commits 
6. Pull requests should always be rebased off of the main branch (unless it is trivial)
7. Pull requests must pass golangci-lint linting (some freedom is given in cases where golangci-lint updates require large changes accross the codebase)

Besides that there are general guidelines:

1. try to make new features optional of config/command flagged.
2. try to add new functionality to phases so that it can be organized and reused across commands
3. try to make phases detect if they need to be run (as opposed to conditionally including the phases) - note that there is currently diversity in this in existing phases
4. try to avoid changes in config syntax; if changes are needed, try to make the changes optional so that the schema isn't broken
5. if you break the schema, bump the version and inclued a migration in your change.

### PR Process

1. ensure that you have a related issue for discussion of contributions at a higher level
2. fork the repository and contribute changes to a branch in the fork 
3. open a Pull Request onto the main repo from the fork branch
4. if you don't get feedback in a timely manner, ping @james-nesbitt on the PR

# PRODENG-3393 — launchpad cannot deploy MKE 3.8 using template from the doc

**Priority:** Unassigned | **Reporter:** Dzmitry Stremkouski | **Customer:** Nordea PreSales

## Problem

Users following the MKE 3.8 doc template cannot deploy because:

1. The doc template omits the `mcr` block entirely, so launchpad uses defaults.
2. In v1.5.15 (when filed), the default MCR version was old (20.10.13) and the
   `get.mirantis.com` install script failed with exit status 1 on Ubuntu 22.04.
3. In v1.5.16, `mcr.version` was removed from `MCRConfig`; the install path
   switched to direct apt package management (`pkg/configurer/ubuntu/ubuntu.go`).
   The failure mode on Ubuntu 22.04 may differ with the new install path.

Related: PRODENG-3342 (same reporter, same customer, same root MCR install failure).

## Root Cause Analysis

### v1.5.15 (filed against)
- `MCRConfig.Version` defaulted to an old pinned value (20.10.13).
- Install used `https://get.mirantis.com/` shell script; exited 1 on Ubuntu 22.04.
- Reporter in PRODENG-3342 noted: "Please disable dpkg script during docker install"
  suggesting a dpkg post-install script conflict (likely Docker default bridge
  network activation conflicting with the environment).

### v1.5.16+ (current)
- `mcr.version` field removed; install is now apt-based (`ubuntu.go:InstallMCR`).
- No `DEBIAN_FRONTEND=noninteractive` is set during apt operations.
- `InstallPackage` installs latest from the configured channel with no version pin.
- The dpkg post-install script issue may still be present (Docker daemon starts on
  install and can fail or conflict in restricted environments).

## Acceptance Criteria

1. `launchpad apply` with a minimal doc-template config (no `mcr` block) on Ubuntu 22.04
   installs MCR successfully.
2. If a dpkg/apt post-install script failure is confirmed, launchpad either suppresses
   the daemon start during install or handles the non-zero exit gracefully.
3. No regression on other Linux distros (EL, SLES).

## Implementation Plan

### Step 1 — Reproduce and confirm current failure mode
- Run `launchpad apply` on a fresh Ubuntu 22.04 host with no `mcr` block.
- Capture the full install log (not just the retry-exhausted summary).
- Determine if `docker-ee` post-install script is the failure point.

### Step 2 — Fix Ubuntu apt install (if dpkg is confirmed)
**File:** `pkg/configurer/ubuntu/ubuntu.go` — `InstallMCR`

Option A (preferred): Set `DEBIAN_FRONTEND=noninteractive` when calling apt and pass
`-o Dpkg::Options::="--force-confold"` to prevent post-install script interaction.

Option B: Disable the Docker daemon auto-start during install:
```bash
echo "exit 101" | sudo tee /usr/sbin/policy-rc.d && sudo chmod +x /usr/sbin/policy-rc.d
# ... install packages ...
sudo rm -f /usr/sbin/policy-rc.d
```
Then explicitly start/enable via `EnableMCR`.

### Step 3 — Fix or update the `InstallPackage` helper
**File:** `pkg/configurer/linux.go` (or ubuntu-specific override)

Confirm `InstallPackage` passes `-y` and `DEBIAN_FRONTEND=noninteractive`.
If not, fix it here so all package installs benefit.

### Step 4 — Doc fix
The doc example for MKE 3.8 must include a valid `mcr` block showing the
correct channel and repoURL. File against the docs team or update in-repo examples.
Note: `mcr.version` is no longer a valid field as of v1.5.16 — any existing docs
that show `mcr.version` must be updated.

### Step 5 — Verification
- Unit test: `pkg/configurer/ubuntu/ubuntu_test.go` — mock exec, assert
  `DEBIAN_FRONTEND=noninteractive` is present in install commands.
- Integration: deploy on Ubuntu 22.04 with a bare (no `mcr` block) config.

## Files in Scope

| File | Change |
|---|---|
| `pkg/configurer/ubuntu/ubuntu.go` | Fix `InstallMCR` — noninteractive apt / policy-rc.d |
| `pkg/configurer/linux.go` | Audit `InstallPackage` for `-y` / noninteractive flags |
| `pkg/configurer/ubuntu/ubuntu_test.go` | Unit tests for install command construction |
| `examples/` or `docs/` | Update example YAML (no `mcr.version`, correct channel) |

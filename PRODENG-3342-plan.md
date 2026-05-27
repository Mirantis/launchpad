# PRODENG-3342 — Cannot deploy MKE 3.8 using templates from the doc

**Priority:** Critical | **Reporter:** Dzmitry Stremkouski | **Customer:** Nordea PreSales

## Problem

Two sequential failures when following the MKE 3.8 doc template with launchpad 1.5.15
on Ubuntu 22.04. PRODENG-3393 is a clean repro of failure #1 on fresh nodes.

---

## Failure 1 — MCR install fails (shared with PRODENG-3393)

Doc template omits the `mcr` block. Launchpad used the `get.mirantis.com` install
script with the old default version (20.10.13), which fails on Ubuntu 22.04 with
exit status 1 after 10 retries on all nodes.

**This failure is being fixed in the PRODENG-3393 worktree.** Once that fix lands,
cherry-pick or rebase this branch on top of it. Do not duplicate the MCR install
fix here.

---

## Failure 2 — MKE bootstrap connection timeout (this worktree's scope)

After manually adding the `mcr` block and retrying, MKE install fails:

```
INFO MKE install: Possible conflict between Kubernetes service CIDR range
     10.96.0.0/16 and default address pool for Swarm overlay networks 10.0.0.0/8
...
FATA failed to apply cluster: failed to apply MKE: phase failure:
     Upgrade MKE components => [ssh] ...: read: connection timed out
```

MKE logs detect the CIDR conflict and warn, but proceed. The install agent then
loses its SSH connection — likely because the Docker daemon restart during MKE
bootstrap tears down the overlay network that the SSH session is riding on,
and the new overlay is in a conflicting address space that disrupts routing.

The reporter's config had:
```yaml
mke:
  installFlags:
    - --pod-cidr 10.0.0.0/16
```

`10.0.0.0/16` is a subnet of Swarm's default overlay pool `10.0.0.0/8` — this
is a direct conflict. Docker itself warns about it.

### Root cause options

**A — User config error, no code change needed.**
`--pod-cidr 10.0.0.0/16` conflicting with `10.0.0.0/8` is a known Docker Swarm
constraint. The doc should warn users to pick a non-overlapping CIDR.

**B — Launchpad should detect and reject this at validation time.**
The `ValidateFacts` or a dedicated validation phase could parse `--pod-cidr` from
`mke.installFlags`, compare it against the Swarm default overlay pool (`10.0.0.0/8`),
and fail fast with a clear error before any install begins.

## Recommendation

Implement option B. A silent connection timeout after 20+ minutes of install is a
poor user experience for a detectable misconfiguration. Fast-fail with a clear message.

## Acceptance Criteria

1. `launchpad apply` with `--pod-cidr` that overlaps `10.0.0.0/8` fails immediately
   at validation with a descriptive error naming the conflict.
2. `launchpad apply` with a non-overlapping `--pod-cidr` (e.g. `192.168.0.0/16`)
   proceeds normally.
3. Configs without `--pod-cidr` are unaffected.

## Implementation Plan

### Step 1 — Add pod-CIDR overlap validation
**File:** `pkg/product/mke/phase/validate_facts.go` (or a new
`pkg/product/mke/phase/validate_config.go`)

- Parse `--pod-cidr` value from `p.Config.Spec.MKE.InstallFlags`.
- Parse the Swarm default overlay pool: `10.0.0.0/8` (constant or from
  `mcr.swarmInstallFlags` if `--default-addr-pool` is set there).
- Use `net.ParseCIDR` + overlap check; fail with actionable message:
  ```
  FATA invalid config: --pod-cidr 10.0.0.0/16 overlaps with the Swarm default
       overlay address pool 10.0.0.0/8; choose a non-overlapping range or set
       mcr.swarmInstallFlags --default-addr-pool to a non-conflicting pool
  ```

### Step 2 — Unit tests
**File:** `pkg/product/mke/phase/validate_facts_test.go` (or new test file)

- Overlapping CIDR → validation error.
- Non-overlapping CIDR → no error.
- No `--pod-cidr` flag → no error.
- Custom `--default-addr-pool` in swarmInstallFlags → validate against that instead.

### Step 3 — Verification
- `make unit-test` passes.
- Manual: `launchpad apply` with conflicting CIDR → immediate clear error.

## Files in Scope

| File | Change |
|---|---|
| `pkg/product/mke/phase/validate_facts.go` | Add pod-CIDR / Swarm overlap check |
| `pkg/product/mke/phase/validate_facts_test.go` | Unit tests for new validation |
| `pkg/product/mke/config/` | Possibly add `Flags.GetValue` helper if not present |

## Dependencies

- MCR install fix (PRODENG-3393) must land first or be cherry-picked in, since
  failure 1 blocks reaching failure 2 in any real test.

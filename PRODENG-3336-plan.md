# PRODENG-3336 — Launchpad uses private IPs for Swarm instead of config addresses

**Priority:** Major | **Reporter:** Dzmitry Stremkouski

## Problem

In stretched/multi-DC environments, hosts have:
- A private interface IP (e.g. `192.168.x.x`) — not routable across DCs
- A floating/public SSH address (e.g. `172.19.x.x`) — routable across DCs

Launchpad unconditionally uses `Metadata.InternalAddress` (resolved from
`privateInterface`) as the Swarm advertise address and join target.
In stretched topologies this means Swarm nodes cannot reach each other and
the cluster fails to form.

Reporter's working workaround:
```bash
docker swarm init --advertise-addr 172.19.121.30 --force-new-cluster
docker swarm join --token <token> --advertise-addr 172.19.124.62 172.19.121.30:2377
```
i.e. using the SSH/floating addresses from the launchpad config.

## Root Cause

`pkg/product/mke/config/host.go`:
```go
func (h *Host) SwarmAddress() string {
    return fmt.Sprintf("%s:%d", h.Metadata.InternalAddress, 2377)
}
```

`InternalAddress` is set in `gather_facts.go` by querying the `privateInterface`
NIC — always a local, possibly non-routable IP.

Additionally, `swarm join` does not set `--advertise-addr` for the joining node at all:
```go
// join_controllers.go / join_workers.go
joinCmd := h.Configurer.DockerCommandf("swarm join --token %s %s",
    token, swarmLeader.SwarmAddress())
```
The joining node will advertise its own auto-detected address, which will again
be the private NIC IP and not reachable across DCs.

## Acceptance Criteria

1. Users can set an explicit `swarmAddress` per host in the launchpad YAML.
2. When set, that address is used for `swarm init --advertise-addr` (leader)
   and `swarm join --advertise-addr` (joining nodes).
3. When not set, behaviour is unchanged (falls back to `InternalAddress`).
4. The join command for both managers and workers passes
   `--advertise-addr=<joining host swarm address>`.

## Implementation Plan

### Step 1 — Add `swarmAddress` field to Host config
**File:** `pkg/product/mke/config/host.go`

```go
// Host struct — add field:
SwarmAddress string `yaml:"swarmAddress,omitempty"`
```

Update `SwarmAddress()` method (rename to avoid collision):
```go
// SwarmAddr returns the address used for Swarm clustering.
// Uses the explicit swarmAddress config field when set,
// otherwise falls back to the discovered InternalAddress.
func (h *Host) SwarmAddr() string {
    addr := h.Metadata.InternalAddress
    if h.SwarmAddress != "" {
        addr = h.SwarmAddress
    }
    return fmt.Sprintf("%s:%d", addr, 2377)
}
```

Note: rename from `SwarmAddress()` to `SwarmAddr()` to avoid the field/method
name collision, OR keep the field as `swarmAddress` yaml but name the struct
field `SwarmAddressOverride`. Pick one and apply consistently.

### Step 2 — Update all SwarmAddress() call sites
**Files:** `pkg/product/mke/phase/init_swarm.go`,
`pkg/product/mke/phase/join_controllers.go`,
`pkg/product/mke/phase/join_workers.go`,
`pkg/product/mke/config/cluster_spec.go`

Run `lsp references` on `SwarmAddress` before touching anything.
Update each call site to use the renamed method.

### Step 3 — Add `--advertise-addr` to join commands
**File:** `pkg/product/mke/phase/join_controllers.go`

```go
joinCmd := h.Configurer.DockerCommandf(
    "swarm join --advertise-addr=%s --token %s %s",
    h.SwarmAddr(), token, swarmLeader.SwarmAddr())
```

**File:** `pkg/product/mke/phase/join_workers.go` — same pattern.

### Step 4 — Update tests
**File:** `pkg/product/mke/config/host_test.go`

- `TestHostSwarmAddress`: add cases for explicit `swarmAddress` override.
- Add case: override takes precedence over `InternalAddress`.
- Add case: empty override falls back to `InternalAddress`.

### Step 5 — Verification
- `make unit-test` passes.
- Manual: deploy with `swarmAddress` set on each host to the SSH address;
  confirm Swarm forms correctly across DCs.

## Files in Scope

| File | Change |
|---|---|
| `pkg/product/mke/config/host.go` | Add `SwarmAddress` field; rename/update `SwarmAddr()` method |
| `pkg/product/mke/config/host_test.go` | Test override behaviour |
| `pkg/product/mke/phase/init_swarm.go` | Update call site |
| `pkg/product/mke/phase/join_controllers.go` | Update call site; add `--advertise-addr` for joining node |
| `pkg/product/mke/phase/join_workers.go` | Update call site; add `--advertise-addr` for joining node |
| `pkg/product/mke/config/cluster_spec.go` | Update call site if present |

## Notes

- No migration needed — the field is purely additive and optional.
- The `privateInterface` / `InternalAddress` path is unchanged for users who
  don't need stretched topology.
- Consider adding a validation warning when `SwarmAddress` is set but equals
  `InternalAddress` (no-op override, probably a config mistake).

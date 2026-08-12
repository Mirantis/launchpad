# Swarm overlay address pool

Swarm allocates the subnets for overlay networks — including `ingress` — from its
default address pool. Unless a pool was given when the swarm was created, that
pool is `10.0.0.0/8` with a 24-bit subnet mask, so `ingress` becomes `10.0.0.0/24`.

## The pool cannot be changed after the swarm is created

`--default-addr-pool` is accepted only by `docker swarm init`. It is not part of
the swarm specification that `docker swarm update` modifies, so there is no
command that changes the pool of a running swarm:

```
$ docker swarm update --default-addr-pool 10.10.0.0/16
unknown flag: --default-addr-pool
```

Launchpad passes `spec.mcr.swarmInstallFlags` to `docker swarm init`, which means
those flags take effect only on the run that creates the swarm. On any later run
against an existing cluster they are ignored, and launchpad logs a warning
naming them.

Setting `--default-addr-pool` on a cluster whose swarm already exists therefore
changes nothing on the hosts. Launchpad reports this as a warning:

```
mcr.swarmInstallFlags sets --default-addr-pool 10.0.0.0/16 but the existing swarm
allocates overlay networks from 10.0.0.0/8. A swarm's address pool is fixed when
the swarm is created and cannot be changed on a running cluster, so this setting
has no effect here and the cluster no longer matches its configuration.
```

## Reading the pool a cluster is using

On a manager:

```bash
docker info --format '{{.Swarm.LocalNodeState}}|{{if .Swarm.Cluster}}{{range .Swarm.Cluster.DefaultAddrPool}}{{.}} {{end}}{{end}}'
```

An empty pool list means the swarm was created without `--default-addr-pool` and
is using the `10.0.0.0/8` default. The subnet mask length is
`{{.Swarm.Cluster.SubnetSize}}`.

The `ingress` network is allocated from the pool and confirms it independently:

```bash
docker network inspect ingress --format '{{range .IPAM.Config}}{{.Subnet}}{{end}}'
```

## Overlap with the Kubernetes pod CIDR

`--pod-cidr` in `spec.mke.installFlags` must not overlap the Swarm pool.
An overlap can leave the container runtime with a broken network configuration
during MKE bootstrap, which drops the SSH connection and surfaces as a
connection timeout after 20 or more minutes.

Launchpad checks this in the `Validate Facts` phase, and what it does depends on
whether the conflict is still avoidable:

| Cluster state | Pool compared against | On overlap |
|---|---|---|
| No swarm yet | `spec.mcr.swarmInstallFlags`, or `10.0.0.0/8` | Fails. Choose a non-overlapping `--pod-cidr`, or set `--default-addr-pool`, which will be applied. |
| Swarm exists | The pool read from the swarm | Warns and continues. The pool cannot be changed, so failing would only block upgrades of clusters already running this way. |

## Changing the pool on an existing cluster

There is no non-destructive procedure. The swarm must be dissolved and
re-created, which **destroys every overlay network and every service running on
them**, including `ingress`. Plan a maintenance window.

If the goal is only to resolve a pod CIDR overlap, changing `--pod-cidr` instead
is far less disruptive and is the recommended option.

To change the pool:

1. Record the current configuration — `docker network ls`, and
   `docker network inspect` for each overlay network you will need to re-create.
2. Stop workloads that depend on overlay networking.
3. On every node, leave the swarm: `docker swarm leave --force`.
4. Set the desired pool in the cluster configuration:

   ```yaml
   spec:
     mcr:
       swarmInstallFlags:
         - --default-addr-pool=10.10.0.0/16
         - --default-addr-pool-mask-length=24
   ```

5. Run `launchpad apply`. With no swarm present, `InitSwarm` creates one and the
   flags are applied.
6. Confirm the new pool with the `docker info` command above, and check that
   `ingress` has been allocated from it.
7. Re-create the overlay networks and redeploy the workloads recorded in step 1.

`--default-addr-pool` may be repeated to give the swarm more than one pool.
Launchpad validates `--pod-cidr` against all of them.

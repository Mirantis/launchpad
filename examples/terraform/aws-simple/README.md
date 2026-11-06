# Infrastructure provision and install


## Provisioning


## Distribution installation

### Launchpad

#### Network exposure — `admin_cidrs`

`admin_cidrs` lists the CIDRs allowed to reach the cluster from outside it:
SSH, WinRM over HTTPS, and the MKE/MSR and Kubernetes API ports.

It defaults to `["0.0.0.0/0"]` — open to the whole internet. That is a real
risk, and it is the exposure this example has always had; it is the default so
that an existing `terraform.tfvars` keeps working and a first run needs no
extra arguments. Narrow it whenever you can:

```hcl
admin_cidrs = ["198.51.100.17/32", "203.0.113.0/24"]
```

Node-to-node traffic does not pass through these rules, so narrowing this does
not affect the cluster talking to itself.

#### NodePort exposure — `nodeport_cidrs`

This example installs MKE with `--nodeport-range=32768-35535`, so Kubernetes
NodePort services land in that range on every node. `nodeport_cidrs` lists the
CIDRs allowed to reach it. The range itself is stated once, as
`local.nodeport_range` in `launchpad.tf`, and feeds both the install flag and
the security group rule so the two cannot disagree.

Leaving it unset means "use `admin_cidrs`", which keeps an existing
`terraform.tfvars` working and carries `admin_cidrs`'s exposure with it: at
both defaults the NodePort range is reachable from the whole internet. Set it
when the people who reach your applications are not the people who administer
the cluster — widening `admin_cidrs` to let application clients in would also
hand them SSH and the MKE and Kubernetes APIs:

```hcl
nodeport_cidrs = ["203.0.113.0/24"]
```

The rule covers TCP and UDP, and does not include the VPC CIDR: nothing in
this example load-balances the NodePort range, and node-to-node traffic is
already covered by the cluster-internal rule.

On `rhel_9` and `sles_15` nodes the host firewall is a second gate. The
`user_data` in `terraform.tfvars.template` opens 2377, 7946, 4789 and 10250
but not the NodePort range, so `firewall-cmd` still blocks it there whatever
the security group says. Add the range to that `user_data` if you need
NodePort services on those platforms.
The `ubuntu_22.04` `user_data` omits the range in exactly the same way — it
opens 2377, 7946, 10250/tcp and 7946, 4789/udp through `ufw` and nothing else —
so the same gate applies on Ubuntu whenever `ufw` is active in that AMI, which
is unverified here because confirming it needs a running AWS cluster.

#### Credentials

The MKE admin password (`launchpad.mke_connect.password`) and
`windows_password` are ordinary terraform variables. Setting them in
`terraform.tfvars` is all that is required, and the generated
`launchpad_yaml` output is usable as-is.

To keep a password off disk, use terraform's own environment mechanism —
terraform reads any input variable from `TF_VAR_<name>`:

```sh
export TF_VAR_launchpad='{"drain":false,"mcr_channel":"stable-25.0.14","mke_version":"3.8.8","msr_version":"2.9.28","mke_connect":{"username":"admin","password":"...","insecure":true},"skip_create":false,"skip_destroy":true}'
export TF_VAR_windows_password='...'
```

The MKE password is nested inside the `launchpad` object, so `TF_VAR_launchpad`
carries that whole object as JSON. Either route works; the environment is an
option for keeping secrets out of a file, not a requirement for running the
example.


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


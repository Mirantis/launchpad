# Infrastructure provision and install

## Provisioning

- This terraform configuration creates infrastructure required for running Launchpad

## Installation

- This configuration uses Launchpad terraform provider for installing MCR, MKE and MSR 2.9x on the provisioned infrstructure
- After installing MKE, the configuration uses MKE terraform provider to create a kubernetes context to install kubernetes workloads
- After creating a kubernetes context the same is used for installing AWS cloud controller manager and AWS EBS CSI driver
- The content of test.tf file is commented out and the resources in that file can be used testing the functionality of AWS cloud controller manager and AWS EBS CSI driver

## Inputs worth knowing about

### Network exposure — `admin_cidrs`

`admin_cidrs` lists the CIDRs allowed to reach the cluster from outside it:
SSH, WinRM over HTTPS, and the MKE and Kubernetes API ports.

It defaults to `["0.0.0.0/0"]` — open to the whole internet. That is a real
risk, and it is the exposure this example has always had; it is the default so
that an existing `terraform.tfvars` keeps working and a first run needs no
extra arguments. Narrow it whenever you can:

```hcl
admin_cidrs = ["198.51.100.17/32", "203.0.113.0/24"]
```

Node-to-node traffic does not pass through these rules, so narrowing this does
not affect the cluster talking to itself.

### Credentials

The MKE admin password (`launchpad.mke_connect.password`) and
`windows_password` are ordinary terraform variables. Setting them in
`terraform.tfvars` is all that is required, and the generated
`launchpad_yaml` output is usable as-is.

To keep a password off disk, use terraform's own environment mechanism —
terraform reads any input variable from `TF_VAR_<name>`:

```sh
export TF_VAR_launchpad='{"drain":false,"mcr_version":"25.0.13","mke_version":"3.8.8","msr_version":"","mke_connect":{"username":"admin","password":"...","insecure":false},"skip_create":false,"skip_destroy":true}'
export TF_VAR_windows_password='...'
```

The MKE password is nested inside the `launchpad` object, so `TF_VAR_launchpad`
carries that whole object as JSON. Either route works; the environment is an
option for keeping secrets out of a file, not a requirement for running the
example.
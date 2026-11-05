
variable "aws" {
  description = "AWS configuration"
  type = object({
    region = string
  })
  default = {
    region = "us-east-1"
  }
}

variable "name" {
  description = "stack/cluster name, used in labelling across the stack."
  type        = string
}

# ===  Networking ===
variable "network" {
  description = "Network configuration"
  type = object({
    cidr               = string
    enable_nat_gateway = optional(bool, false)
    enable_vpn_gateway = optional(bool, false)
    tags               = optional(map(string), {})
  })
  default = {
    enable_nat_gateway = false
    enable_vpn_gateway = false
    cidr               = "172.31.0.0/16"
    tags               = {}
  }
}

# === subnets ===
variable "subnets" {
  description = "The subnets configuration"
  type = map(object({
    cidr       = string
    nodegroups = list(string)
    private    = optional(bool, false)
  }))
  default = {}
}

# === Machines ===
variable "nodegroups" {
  description = "A map of machine group definitions"
  type = map(object({
    role        = string
    platform    = string
    type        = string
    count       = optional(number, 1)
    volume_size = optional(number, 100)
    public      = optional(bool, true)
    // Takes precedence over the platform's own coded-default user_data
    // (see provision.tf: nodegroups_wplatform), with one exception: winrm
    // platforms have a fixed requirement -- an Administrator password
    // reset and WinRM-over-HTTPS listener setup -- that launchpad cannot
    // connect without, so on those platforms this is appended after that
    // required setup rather than replacing it.
    user_data = optional(string, "")
  }))
}

variable "extra_tags" {
  description = "Extra tags that will be added to all provisioned resources, where possible."
  type        = map(string)
  default     = {}
}

variable "admin_cidrs" {
  description = <<-EOT
    CIDRs allowed to reach the cluster from outside it: SSH, WinRM, and the
    MKE/MSR and Kubernetes API ports.  Required and intentionally without a
    default: the previous rule set accepted every port from 0.0.0.0/0, and a
    default would reintroduce that silently.  Supply the narrowest set that
    covers the machine running launchpad and the operators who need the UI.
  EOT
  type        = list(string)
  validation {
    condition     = length(var.admin_cidrs) > 0
    error_message = "admin_cidrs must list at least one CIDR."
  }
}

variable "windows_password" {
  description = <<-EOT
    Password set on the local Administrator account of Windows nodes and used
    by launchpad for WinRM.  Required when any nodegroup runs a Windows
    platform; leave unset for Linux-only stacks.

    The validation below references var.nodegroups, which Terraform supports
    from 1.9 onwards.
  EOT
  type        = string
  sensitive   = true
  default     = null

  // Without this, a stack with Windows nodegroups and no windows_password
  // generated a launchpad.yaml carrying "password:" with an empty value, and
  // launchpad then authenticated as Administrator with no password. Failing
  // here names the variable at plan time instead.
  //
  // The whole requirement is gated on a Windows nodegroup being present, so a
  // Linux-only stack may leave the variable unset or pass an empty string --
  // neither reaches a host that would authenticate with it.
  validation {
    condition = (
      !anytrue([for ng in var.nodegroups : startswith(ng.platform, "windows_")])
      || (var.windows_password != null && length(var.windows_password) > 0)
    )
    error_message = "windows_password must be set to a non-empty value when any nodegroup uses a windows_ platform: it is the local Administrator password the nodes are configured with and the credential launchpad uses for WinRM. Leave it unset only for Linux-only stacks."
  }
}

variable "ssh_pk_location" {
  description = "Location of the ssh private key"
  type        = string
  default     = ""
}

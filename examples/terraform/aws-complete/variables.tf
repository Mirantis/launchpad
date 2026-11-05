
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
    user_data   = optional(string, "")
  }))
}

variable "extra_tags" {
  description = "Extra tags that will be added to all provisioned resources, where possible."
  type        = map(string)
  default     = {}
}

variable "admin_cidrs" {
  description = <<-EOT
    CIDRs allowed to reach the cluster from outside it: SSH, WinRM over HTTPS,
    and the MKE and Kubernetes API ports.  Required and intentionally without
    a default: the previous rule set accepted those ports from 0.0.0.0/0, and
    a default would reintroduce that silently.
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
  validation {
    condition = (
      var.windows_password == null
      ? !anytrue([for ng in var.nodegroups : startswith(ng.platform, "windows_")])
      : length(var.windows_password) > 0
    )
    error_message = "windows_password must be set to a non-empty value when any nodegroup uses a windows_ platform: it is the local Administrator password the nodes are configured with and the credential launchpad uses for WinRM. Leave it unset only for Linux-only stacks."
  }
}

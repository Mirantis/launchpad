
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

variable "windows_password" {
  description = "Password to use with windows & winrm"
  type        = string
  default     = ""
}

variable "ssh_pk_location" {
  description = "Location of the ssh private key"
  type        = string
  default     = ""
}

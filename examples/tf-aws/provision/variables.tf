
variable "name" {
  description = "cluster/stack name used for identification"
  type        = string
}

# ===  Networking ===

variable "cidr" {
  description = "cidr for the stack internal network"
  type        = string
  default     = "172.31.0.0/16"
}

variable "public_subnet_count" {
  description = "How many public subnets to create. Subnets will be spread accross region AZs"
  type        = number
  default     = 3
}

variable "private_subnet_count" {
  description = "How many private subnets to create. Subnets will be spread accross region AZs"
  type        = number
  default     = 3
}

variable "tags" {
  description = "tags to be applied to created resources"
  type        = map(string)
}

variable "enable_nat_gateway" {
  description = "Should a NAT gateway be included in the cluster"
  type        = bool
  default     = false
}

variable "enable_vpn_gateway" {
  description = "Should a VPN gateway be included in the cluster"
  type        = bool
  default     = false
}

# === Machines ===

variable "nodegroups" {
  description = "A map of machine group definitions"
  type = map(object({
    ami              = string
    keypair_id       = string
    type             = string
    count            = number
    root_device_name = string
    volume_size      = number
    role             = string
    public           = bool
    user_data        = string
  }))
  default = {}
}

# === Firewalls ===

variable "securitygroups" {
  description = "VPC Network Security group configuration"
  type = map(object({
    description = string
    nodegroups  = list(string) # which nodegroups should get attached to the sg?

    ingress_ipv4 = list(object({
      description = string
      from_port   = number
      to_port     = number
      protocol    = string
      cidr_blocks = list(string)
      self        = bool
    }))
    egress_ipv4 = list(object({
      description = string
      from_port   = number
      to_port     = number
      protocol    = string
      cidr_blocks = list(string)
      self        = bool
    }))
  }))
  default = {}
}

# === Ingresses ===

variable "ingresses" {
  description = "Configure ingress (ALB) for specific nodegroup roles"
  type = map(object({
    description = string
    nodegroups  = list(string) # which nodegroups should get attached to the ingress

    routes = map(object({
      port_incoming = number
      port_target   = number
      protocol      = string
    }))
  }))
  default = {}
}

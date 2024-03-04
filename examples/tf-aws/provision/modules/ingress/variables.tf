
variable "stack" {
  description = "Stack name used for labelling resources"
  type        = string
}

variable "name" {
  description = "Names identifier for the ingress"
  type        = string
}

variable "vpc_id" {
  description = "ID for the vpc to attach to"
  type        = string
}

variable "subnet_ids" {
  description = "List of subnet ids to include"
  type        = list(string)
}

variable "instance_ids" {
  description = "Instance/Machines to target with the ingress"
  type        = list(string)
}

variable "routes" {
  description = "What traffic should the ingress handle"
  type = map(object({
    port_incoming = number
    port_target   = number
    protocol      = string
  }))
}

variable "tags" {
  description = "tags to be applied to created resources"
  type        = map(string)
}

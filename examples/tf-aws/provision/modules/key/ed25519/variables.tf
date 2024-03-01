
variable "name" {
  description = "What label to use for the keypair"
  type        = string
}

variable "tags" {
  description = "Tags to apply to the resources being generated"
  type        = map(string)
  default     = {}
}

#
# SSH keypair — supports both ed25519 (default) and rsa (required for Windows nodes).
#

variable "ssh_key_algorithm" {
  description = "Algorithm for the generated SSH keypair. Must be 'rsa' or 'ed25519'. Use 'rsa' when Windows nodes are present."
  type        = string
  default     = "ed25519"
  validation {
    condition     = contains(["rsa", "ed25519"], var.ssh_key_algorithm)
    error_message = "ssh_key_algorithm must be 'rsa' or 'ed25519'."
  }
}

resource "tls_private_key" "this" {
  algorithm = var.ssh_key_algorithm == "rsa" ? "RSA" : "ED25519"
  rsa_bits  = var.ssh_key_algorithm == "rsa" ? 4096 : null
}

resource "aws_key_pair" "this" {
  key_name   = "${var.name}-common"
  public_key = tls_private_key.this.public_key_openssh
  tags       = local.tags
}

locals {
  pk_path = var.ssh_pk_location != "" ? join("/", [var.ssh_pk_location, "${var.name}-common.pem"]) : "./ssh-keys/${var.name}-common.pem"
}

resource "local_sensitive_file" "ssh_private_key" {
  content              = tls_private_key.this.private_key_openssh
  filename             = local.pk_path
  file_permission      = "0600"
  directory_permission = "0700"
}

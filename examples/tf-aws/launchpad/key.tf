#
# We could use multiple keys for this stack if needed
#

module "key" {
  source = "terraform-mirantis-modules/provision-aws/mirantis//modules/key/ed25519"

  name = "${var.name}-common"
  tags = local.tags
}

locals {
  pk_path = var.ssh_pk_location != "" ? join("/", [var.ssh_pk_location, "${var.name}-common.pem"]) : "./ssh-keys/${var.name}-common.pem"
}

resource "local_sensitive_file" "ssh_private_key" {
  content              = module.key.private_key
  filename             = local.pk_path
  file_permission      = "0600"
  directory_permission = "0700"
}

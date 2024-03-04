#
# We could use multiple keys for this stack if needed
#

module "key" {
  source = "../provision/modules/key/ed25519"

  name = "${var.name}-common.pem"
  tags = local.tags
}

locals {
  pk_path = var.ssh_pk_location != "" ? join("/", [var.ssh_pk_location, module.key.name]) : "./ssh-keys/${module.key.name}"
}

resource "local_sensitive_file" "ssh_private_key" {
  content              = module.key.private_key
  filename             = local.pk_path
  file_permission      = "0777"
  directory_permission = "0777"
}

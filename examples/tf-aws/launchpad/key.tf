#
# We could use multiple keys for this stack if needed
#

module "key" {
  source = "terraform-mirantis-modules/provision-aws/mirantis//modules/key/ed25519"

  name = "${var.name}-common"
  tags = local.tags
}

resource "local_sensitive_file" "ssh_private_key" {
  content              = module.key.private_key
  filename             = join("/", [var.ssh_pk_location, "${var.name}-common.pem"])
  file_permission      = "0600"
  directory_permission = "0700"
}

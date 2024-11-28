#
# We could use multiple keys for this stack if needed
#

module "key" {
  source = "terraform-mirantis-modules/provision-aws/mirantis//modules/key/ed25519"

  name = "${var.name}-commonkey"
  tags = local.tags
}

resource "local_sensitive_file" "ssh_private_key" {
  content              = module.key.private_key
  filename             = "ed25519.key"
  file_permission      = "0600"
  directory_permission = "0644"
}

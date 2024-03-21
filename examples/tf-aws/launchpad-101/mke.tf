
provider "mke" {
  endpoint          = "https://${local.MKE_URL}"
  username          = var.launchpad.mke_connect.username
  password          = var.launchpad.mke_connect.password
  unsafe_ssl_client = var.launchpad.mke_connect.insecure
}

output "mke_connect" {
  description = "Connection information for connecting to MKE"
  sensitive   = true
  value = {
    host     = "https://${local.MKE_URL}"
    username = var.launchpad.mke_connect.username
    password = var.launchpad.mke_connect.password
    insecure = var.launchpad.mke_connect.insecure
  }
}

resource "mke_clientbundle" "admin" {
  label = "terraform-admin"

  depends_on = [
    launchpad_config.cluster
  ]
}


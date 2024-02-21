
resource "time_static" "now" {}

locals {

  // build some tags for all things
  tags = merge(
    { # excludes kube-specific tags
      "stack"   = var.name
      "created" = time_static.now.rfc3339
    },
    var.extra_tags
  )

}

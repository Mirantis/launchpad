
resource "time_static" "now" {}

locals {
  kube_tags = {
    "kubernetes.io/cluster/${var.name}" = "owned"
  }
  common_tags = {
    "stack"   = var.name
    "created" = time_static.now.rfc3339
  }
}

locals {

  // build some tags for all things
  tags = merge(
    local.common_tags,
    var.extra_tags,
    local.kube_tags,
  )

}


module "aws_ccm" {
  source        = "./modules/aws-ccm"
  iam_role_name = aws_iam_role.common_role.name
  cluster_name  = var.name
}

module "aws_ebs_csi_driver" {
  source        = "./modules/ebs-csi-driver"
  iam_role_name = aws_iam_role.common_role.name
}

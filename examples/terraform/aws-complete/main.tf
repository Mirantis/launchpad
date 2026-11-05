
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
  source = "./modules/aws-ccm"
  // The control-plane policy goes to managers only; the worker policy, which
  // is ECR pull access, is needed by every node that runs containers.
  control_plane_role_name = aws_iam_role.manager_role.name
  worker_role_names       = [aws_iam_role.manager_role.name, aws_iam_role.worker_role.name]
  cluster_name            = var.name
}

module "aws_ebs_csi_driver" {
  source = "./modules/ebs-csi-driver"
  // Attached to both roles, as it was to the single role before: the CSI
  // controller runs on a manager and the node plugin runs everywhere.
  iam_role_names = [aws_iam_role.manager_role.name, aws_iam_role.worker_role.name]
}

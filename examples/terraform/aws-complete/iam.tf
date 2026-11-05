// Manager and worker nodes carry separate roles so that the cloud-controller
// permissions, which only the control plane uses, are not held by every
// worker in the cluster.  Both roles share the same trust policy.
resource "aws_iam_role" "manager_role" {
  name               = "manager-iam-role-${var.name}"
  path               = "/"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_iam_role" "worker_role" {
  name               = "worker-iam-role-${var.name}"
  path               = "/"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_instance_profile" "manager_profile" {
  name = "manager-instance-profile-${var.name}"
  role = aws_iam_role.manager_role.name
}

resource "aws_iam_instance_profile" "worker_profile" {
  name = "worker-instance-profile-${var.name}"
  role = aws_iam_role.worker_role.name
}

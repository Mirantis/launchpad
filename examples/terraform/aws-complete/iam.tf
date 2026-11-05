// Manager and worker nodes carry separate roles so that the cloud-controller
// permissions, which only the control plane uses, are not held by every
// worker in the cluster.  Both roles share the same trust policy.
//
// APPLYING THIS TO AN EXISTING STACK REPLACES ITS IAM RESOURCES. The previous
// common_role, common_profile and their policy attachments are destroyed and
// the new profiles are attached to the launch templates, not to instances
// that are already running: provision-aws does not refresh instances on a
// launch-template change, so nodes provisioned before this change keep
// running with an instance profile whose role no longer exists, and the
// cloud controller manager and EBS CSI driver on them lose their permissions
// until the nodes are replaced. Replace the nodes, or recreate the stack,
// rather than re-applying in place against a live cluster.
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

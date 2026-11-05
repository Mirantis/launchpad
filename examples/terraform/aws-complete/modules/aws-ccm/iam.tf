// The AWS cloud controller manager's permissions are split into three
// statements, following the shape of the policy published by the upstream
// cloud-provider-aws project:
//
//   - discovery calls, which AWS does not support resource-level permissions
//     for at all, so they stay at Resource "*";
//   - calls that create new resources, which cannot be constrained by the tags
//     of a resource that does not exist yet;
//   - calls that mutate or delete existing resources, which are constrained to
//     resources carrying this cluster's ownership tag.
//
// The cluster tag is the one the stack applies to everything it provisions
// (see local.kube_tags in the example root), which is also the tag the cloud
// controller manager puts on the resources it creates.
locals {
  cluster_tag_key = "aws:ResourceTag/kubernetes.io/cluster/${var.cluster_name}"
}

data "aws_iam_policy_document" "control_plane" {
  // Discovery. No resource-level permissions exist for these actions.
  statement {
    sid    = "Discovery"
    effect = "Allow"
    actions = [
      "autoscaling:DescribeAutoScalingGroups",
      "autoscaling:DescribeLaunchConfigurations",
      "autoscaling:DescribeTags",
      "ec2:DescribeAvailabilityZones",
      "ec2:DescribeInstances",
      "ec2:DescribeRegions",
      "ec2:DescribeRouteTables",
      "ec2:DescribeSecurityGroups",
      "ec2:DescribeSubnets",
      "ec2:DescribeVolumes",
      "ec2:DescribeVpcs",
      "elasticloadbalancing:DescribeListeners",
      "elasticloadbalancing:DescribeLoadBalancerAttributes",
      "elasticloadbalancing:DescribeLoadBalancerPolicies",
      "elasticloadbalancing:DescribeLoadBalancers",
      "elasticloadbalancing:DescribeTargetGroups",
      "elasticloadbalancing:DescribeTargetHealth",
      "kms:DescribeKey",
    ]
    resources = ["*"]
  }

  // Creation. The resource does not exist when the call is made, so it cannot
  // carry the cluster tag yet; the CCM tags what it creates immediately after.
  statement {
    sid    = "CreateClusterResources"
    effect = "Allow"
    actions = [
      "ec2:CreateSecurityGroup",
      "ec2:CreateTags",
      "ec2:CreateVolume",
      "elasticloadbalancing:AddTags",
      "elasticloadbalancing:CreateListener",
      "elasticloadbalancing:CreateLoadBalancer",
      "elasticloadbalancing:CreateLoadBalancerListeners",
      "elasticloadbalancing:CreateLoadBalancerPolicy",
      "elasticloadbalancing:CreateTargetGroup",
      "iam:CreateServiceLinkedRole",
    ]
    resources = ["*"]
  }

  // Mutation of resources this cluster owns.
  statement {
    sid    = "MutateClusterResources"
    effect = "Allow"
    actions = [
      "ec2:AttachVolume",
      "ec2:AuthorizeSecurityGroupIngress",
      "ec2:CreateRoute",
      "ec2:DeleteRoute",
      "ec2:DeleteSecurityGroup",
      "ec2:DeleteVolume",
      "ec2:DetachVolume",
      "ec2:ModifyInstanceAttribute",
      "ec2:ModifyVolume",
      "ec2:RevokeSecurityGroupIngress",
      "elasticloadbalancing:ApplySecurityGroupsToLoadBalancer",
      "elasticloadbalancing:AttachLoadBalancerToSubnets",
      "elasticloadbalancing:ConfigureHealthCheck",
      "elasticloadbalancing:DeleteListener",
      "elasticloadbalancing:DeleteLoadBalancer",
      "elasticloadbalancing:DeleteLoadBalancerListeners",
      "elasticloadbalancing:DeleteTargetGroup",
      "elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
      "elasticloadbalancing:DeregisterTargets",
      "elasticloadbalancing:DetachLoadBalancerFromSubnets",
      "elasticloadbalancing:ModifyListener",
      "elasticloadbalancing:ModifyLoadBalancerAttributes",
      "elasticloadbalancing:ModifyTargetGroup",
      "elasticloadbalancing:RegisterInstancesWithLoadBalancer",
      "elasticloadbalancing:RegisterTargets",
      "elasticloadbalancing:SetLoadBalancerPoliciesForBackendServer",
      "elasticloadbalancing:SetLoadBalancerPoliciesOfListener",
    ]
    resources = ["*"]

    condition {
      test     = "StringEquals"
      variable = local.cluster_tag_key
      // "owned" is what this stack and the CCM apply; "shared" is accepted so
      // a cluster placed in pre-existing tagged infrastructure still works.
      values = ["owned", "shared"]
    }
  }
}

resource "aws_iam_policy" "control_plane_policy" {
  name        = "control-plane-policy-${var.cluster_name}"
  description = "Iam policy for attaching to the IAM instance profile"
  policy      = data.aws_iam_policy_document.control_plane.json
}

// Managers only: nothing on a worker calls the cloud controller manager.
resource "aws_iam_role_policy_attachment" "attach_control_plane_policy" {
  role       = var.control_plane_role_name
  policy_arn = aws_iam_policy.control_plane_policy.arn
}

data "aws_iam_policy_document" "worker_node" {
  statement {
    sid    = "PullImagesAndDiscoverSelf"
    effect = "Allow"
    actions = [
      "ec2:DescribeInstances",
      "ec2:DescribeRegions",
      "ecr:BatchCheckLayerAvailability",
      "ecr:BatchGetImage",
      "ecr:DescribeRepositories",
      "ecr:GetAuthorizationToken",
      "ecr:GetDownloadUrlForLayer",
      "ecr:GetRepositoryPolicy",
      "ecr:ListImages",
    ]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "worker_node_policy" {
  name        = "workernode-policy-${var.cluster_name}"
  description = "Iam policy for attaching to the IAM instance profile"
  policy      = data.aws_iam_policy_document.worker_node.json
}

resource "aws_iam_role_policy_attachment" "attach_worker_node_policy" {
  for_each = toset(var.worker_role_names)

  role       = each.value
  policy_arn = aws_iam_policy.worker_node_policy.arn
}

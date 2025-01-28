resource "aws_iam_role" "common_role" {
  name               = "common-iam-role-${var.name}"
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

resource "aws_iam_instance_profile" "common_profile" {
  name = "common-instance-profile-${var.name}"
  role = aws_iam_role.common_role.name
}

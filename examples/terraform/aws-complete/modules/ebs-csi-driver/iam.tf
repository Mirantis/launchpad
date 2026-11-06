resource "aws_iam_role_policy_attachment" "attach_aws_ebs_csi_policy" {
  for_each = toset(var.iam_role_names)

  role       = each.value
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
}

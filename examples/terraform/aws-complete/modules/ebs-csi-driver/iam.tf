resource "aws_iam_role_policy_attachment" "attach_aws_ebs_csi_policy" {
  role       = var.iam_role_name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
}

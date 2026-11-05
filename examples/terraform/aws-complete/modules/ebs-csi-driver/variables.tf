variable "iam_role_names" {
  type        = list(string)
  description = "Names of the IAM roles to which the policies required for running AWS EBS CSI are attached. The controller runs on a manager and the node plugin runs on every node, so this normally lists both roles."
}

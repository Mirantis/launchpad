variable "iam_role_name" {
  type        = string
  description = "Name of the IAM role to which the policies required for running AWS CCM are attached"
}

variable "cluster_name" {
  type        = string
  description = "Name of the cluster"
}

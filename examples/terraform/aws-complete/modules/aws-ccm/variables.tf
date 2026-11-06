variable "control_plane_role_name" {
  type        = string
  description = "Name of the IAM role carried by manager nodes, to which the AWS CCM control-plane policy is attached"
}

variable "worker_role_names" {
  type        = list(string)
  description = "Names of the IAM roles that run workloads, to which the worker policy (ECR pull access) is attached. Managers run workloads too, so this normally includes the manager role."
}

variable "cluster_name" {
  type        = string
  description = "Name of the cluster"
}

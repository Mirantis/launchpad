variable "cluster_name" {
  default = "ucp"
}

variable "aws_region" {
  default = "eu-north-1"
}

variable "vpc_cidr" {
  default = "172.31.0.0/16"
}

variable "admin_password" {
  default = "ucp-ftw!!!"
}


variable "master_count" {
  default = 1
}

variable "worker_count" {
  default = 3
}

variable "master_type" {
  default = "m5.large"
}

variable "worker_type" {
  default = "m5.large"
}

variable "master_volume_size" {
  default = 100
}

variable "worker_volume_size" {
  default = 100
}


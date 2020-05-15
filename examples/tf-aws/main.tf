provider "aws" {
  region = var.aws_region
}

module "vpc" {
  source       = "./modules/vpc"
  cluster_name = var.cluster_name
  host_cidr    = var.vpc_cidr
}

module "common" {
  source       = "./modules/common"
  cluster_name = var.cluster_name
  vpc_id       = module.vpc.id
}

module "masters" {
  source                = "./modules/master"
  master_count          = var.master_count
  vpc_id                = module.vpc.id
  cluster_name          = var.cluster_name
  subnet_ids            = module.vpc.public_subnet_ids
  security_group_id     = module.common.security_group_id
  image_id              = module.common.image_id
  kube_cluster_tag      = module.common.kube_cluster_tag
  ssh_key               = var.cluster_name
  instance_profile_name = module.common.instance_profile_name
}

module "workers" {
  source                = "./modules/worker"
  worker_count          = var.worker_count
  vpc_id                = module.vpc.id
  cluster_name          = var.cluster_name
  subnet_ids            = module.vpc.public_subnet_ids
  security_group_id     = module.common.security_group_id
  image_id              = module.common.image_id
  kube_cluster_tag      = module.common.kube_cluster_tag
  ssh_key               = var.cluster_name
  instance_profile_name = module.common.instance_profile_name
  worker_type           = var.worker_type
}


output "ucp_cluster" {
  value = {
    ucp = {
      installFlags: [
        "--admin-username=admin",
        "--admin-password=${var.admin_password}",
        "--cloud-provider=aws",
        "--default-node-orchestrator=kubernetes",
        "--san=${module.masters.lb_dns_name}",
      ]
    }
    hosts = [
      for host in concat(module.masters.machines, module.workers.machines) : {
        address = host.public_ip
        user    = "ubuntu"
        role    = host.tags["Role"]
        privateInterface = "ens5"
        sshKeyPath = "./ssh_keys/${var.cluster_name}.pem"
      }
    ]
  }
}

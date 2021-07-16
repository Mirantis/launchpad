locals {
  managers = [
    for ip in module.masters.public_ips : {
      ssh: {
        address = ip
        user    		= "ubuntu"
        keyPath 	= "./ssh_keys/${var.cluster_name}"
        port			 	= 22
      }
      role    = "manager"
      privateInterface = "ens3"
      mcrConfig: {
        bip: var.docker_int_net
        default-address-pools: {
          base: var.docker_default_address_pool
        }
      }
      environment : {
        http_proxy = var.http_proxy
        HTTPS_PROXY = var.https_proxy
      }
    }
  ]
  workers = [
    for ip in module.workers.public_ips : {
      ssh: {
        address = ip
        user    		= "ubuntu"
        keyPath 	= "./ssh_keys/${var.cluster_name}"
        port    		= 22
      }
      role    = "worker"
      privateInterface = "ens3"
      mcrConfig: {
        bip: var.docker_int_net
        default-address-pools: {
          base: var.docker_default_address_pool
        }
      }
      environment : {
        http_proxy = var.http_proxy
        HTTPS_PROXY = var.https_proxy
      }
    }
  ]
  launchpad_tmpl = {
    apiVersion = "launchpad.mirantis.com/mke/v1.3"
    kind = "mke"
    metadata = {
      name = "mkecluster"
    }
    spec = {
      mke = {
        version = var.docker_enterprise_version
        imageRepo = var.docker_image_repo
        adminUsername = "admin"
        adminPassword = var.admin_password
        installFlags: [
          "--default-node-orchestrator=kubernetes",
          "--san=${module.masters.lb_ip}",
        ]
        cloud = {
          provider = "openstack"
          configFile = var.provider_config_file_path
        }
      }
      mcr = {
        version = var.docker_engine_version
        channel = "stable"
        repoURL = "https://repos.mirantis.com"
      }
      hosts = concat(local.managers, local.workers)
    }
  }
}

output "mke_cluster" {
  value = yamlencode(local.launchpad_tmpl)
}

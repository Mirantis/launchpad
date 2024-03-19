// locals calculated before the provision run
locals {
  // combine the nodegroup definition with the platform data
  nodegroups_wplatform = { for k, ngd in var.nodegroups : k => merge(ngd, local.platforms_with_ami[ngd.platform]) }
}

# PROVISION MACHINES/NETWORK
module "provision" {
  source = "terraform-mirantis-modules/provision-aws/mirantis"

  name        = var.name
  common_tags = local.tags
  network     = var.network


  // pass in a mix of nodegroups with the platform information
  nodegroups = { for k, ngd in local.nodegroups_wplatform : k => {
    ami : ngd.ami
    count : ngd.count
    type : ngd.type
    keypair_id : module.key.keypair_id
    root_device_name : ngd.root_device_name
    volume_size : ngd.volume_size
    role : ngd.role
    public : ngd.public
    user_data : ngd.user_data
  } }

  // ingress/lb (should likely merge with an input to allow more flexibility
  ingresses = local.launchpad_ingresses # see launchpad.tf

  // firewall rules (should likely merge with an input to allow more flexibility
  securitygroups = local.launchpad_securitygroups # see launchpad.tf
}

// locals calculated after the provision module is run, but before installation using launchpad
locals {
  // combine each node-group & platform definition with the provisioned nodes
  nodegroups = { for k, ngp in local.nodegroups_wplatform : k => merge({ "name" : k }, ngp, module.provision.nodegroups[k]) }
  ingresses  = { for k, i in local.launchpad_ingresses : k => merge({ "name" : k }, i, module.provision.ingresses[k]) }
}

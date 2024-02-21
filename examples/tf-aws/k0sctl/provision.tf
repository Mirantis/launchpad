// locals calculated before the provision run
locals {
  // combine the nodegroup definition with the platform data
  nodegroups_wplatform = { for k, ngd in var.nodegroups : k => merge(ngd, local.platforms_with_ami[ngd.platform]) }
}

# PROVISION MACHINES/NETWORK
module "provision" {
  source = "../provision"

  name = var.name
  tags = local.tags

  cidr                 = var.network.cidr
  public_subnet_count  = 1
  private_subnet_count = 0
  enable_vpn_gateway   = false
  enable_nat_gateway   = false

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
  ingresses = local.k0s_ingresses # see k0sctl.tf

  // firewall rules (should likely merge with an input to allow more flexibility
  securitygroups = local.k0s_securitygroups # see k0sctl.tf
}

// locals calculated after the provision module is run, but before installation using launchpad
locals {
  // combine each node-group & platform definition with the provisioned nodes
  nodegroups = { for k, ngp in local.nodegroups_wplatform : k => merge({ "name" : k }, ngp, module.provision.nodegroups[k]) }
  ingresses  = { for k, i in local.k0s_ingresses : k => merge({ "name" : k }, i, module.provision.ingresses[k]) }
}

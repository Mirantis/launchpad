// locals calculated before the provision run
locals {
  // combine the nodegroup definition with the platform data
  nodegroups_wplatform = { for k, ngd in var.nodegroups : k => merge(ngd, local.platforms_with_ami[ngd.platform]) }
}

# PROVISION MACHINES/NETWORK
module "provision" {
  source = "terraform-mirantis-modules/provision-aws/mirantis"

  name = var.name
  network = {
    cidr                 = var.network.cidr
    public_subnet_count  = 1
    private_subnet_count = 0
    enable_vpn_gateway   = false
    enable_nat_gateway   = false
    tags                 = local.tags
  }

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
    instance_profile_name : aws_iam_instance_profile.common_profile.name
    tags : local.tags
  } }

  // ingress/lb (should likely merge with an input to allow more flexibility
  ingresses = local.launchpad_ingresses # see launchpad.tf

  // firewall rules (should likely merge with an input to allow more flexibility
  securitygroups = merge(local.launchpad_securitygroups, local.common_security_groups) # see launchpad.tf and sg.tf
}

// locals calculated after the provision module is run, but before installation using launchpad
locals {
  // combine each node-group & platform definition with the provisioned nodes
  nodegroups = { for k, ngp in local.nodegroups_wplatform : k => merge({ "name" : k }, ngp, module.provision.nodegroups[k]) }
  ingresses  = { for k, i in local.launchpad_ingresses : k => merge({ "name" : k }, i, module.provision.ingresses[k]) }
}

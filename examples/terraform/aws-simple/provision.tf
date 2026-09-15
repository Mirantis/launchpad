// locals calculated before the provision run
locals {
  // Combine the nodegroup definition with the platform data. The platform
  // data wins on every key except user_data: a caller-supplied
  // ngd.user_data (e.g. the firewall-cmd/ufw rules in
  // terraform.tfvars.template) takes precedence over the platform's coded
  // default user_data, EXCEPT for winrm platforms, where the platform's
  // default is not merely a convenience default but a fixed requirement --
  // it resets the local Administrator password and stands up the WinRM
  // HTTPS listener launchpad needs in order to connect at all. For those
  // platforms the caller's user_data is appended after the required setup
  // instead of replacing it, so a custom user_data on a Windows nodegroup
  // can't accidentally disable WinRM connectivity.
  nodegroups_wplatform = { for k, ngd in var.nodegroups : k => merge(ngd, local.platforms_with_ami[ngd.platform], {
    user_data = (
      local.platforms_with_ami[ngd.platform].connection == "winrm"
      ? join("\n", compact([local.platforms_with_ami[ngd.platform].user_data, ngd.user_data]))
      : (ngd.user_data != "" ? ngd.user_data : local.platforms_with_ami[ngd.platform].user_data)
    )
  }) }
}

# PROVISION MACHINES/NETWORK
module "provision" {
  source = "terraform-mirantis-modules/provision-aws/mirantis"

  name        = var.name
  common_tags = local.tags
  network     = var.network
  subnets     = var.subnets

  // pass in a mix of nodegroups with the platform information
  nodegroups = { for k, ngd in local.nodegroups_wplatform : k => {
    source_image : {
      ami : ngd.ami
    }
    count : ngd.count
    type : ngd.type
    keypair_id : aws_key_pair.this.key_name
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
  securitygroups = local.launchpad_securitygroups # see launchpad.tf
}

// locals calculated after the provision module is run, but before installation using launchpad
locals {
  // combine each node-group & platform definition with the provisioned nodes
  nodegroups = { for k, ngp in local.nodegroups_wplatform : k => merge({ "name" : k }, ngp, module.provision.nodegroups[k]) }
  ingresses  = { for k, i in local.launchpad_ingresses : k => merge({ "name" : k }, i, module.provision.ingresses[k]) }
}

// variables calculated before ami data is retrieved
locals {
  // find the unique platforms actually used in the node_group_definitions
  unique_used_platforms = distinct([for ngd in var.nodegroups : ngd.platform])

  // platforms defined in the upstream module
  upstream_platform_keys = [for p in local.unique_used_platforms : p if !contains(keys(local.lib_local_platform_definitions), p)]
  // platforms defined locally (not in upstream module)
  local_platform_keys = [for p in local.unique_used_platforms : p if contains(keys(local.lib_local_platform_definitions), p)]

  // local platform AMI definitions (supplements upstream module)
  lib_local_platform_definitions = {
    "ubuntu_24.04" = {
      ami_name   = "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"
      owner      = "099720109477"
      interface  = "eth0"
      connection = "ssh"
      ssh_user   = "ubuntu"
      ssh_port   = 22
    }
    // TODO: remove once terraform-mirantis-modules/terraform-mirantis-provision-aws v0.1.6
    // is picked up by the Terraform Registry and the module source is updated to >= v0.1.6.
    // ubuntu_22.04_fips was added upstream in PR #21 (released in v0.1.6).
    "ubuntu_22.04_fips" = {
      ami_name   = "ubuntu-pro-fips-updates-server/images/hvm-ssd/ubuntu-jammy-22.04-amd64-pro-fips-updates-server-*"
      owner      = "099720109477"
      interface  = "ens5"
      connection = "ssh"
      ssh_user   = "ubuntu"
      ssh_port   = 22
    }
  }
}

module "platform" {
  count  = length(local.upstream_platform_keys)
  source = "terraform-mirantis-modules/provision-aws/mirantis//modules/platform"

  platform_key     = local.upstream_platform_keys[count.index]
  windows_password = var.windows_password
}

data "aws_ami" "local" {
  for_each = { for p in local.local_platform_keys : p => local.lib_local_platform_definitions[p] }

  most_recent = true
  owners      = [each.value.owner]

  filter {
    name   = "name"
    values = [each.value.ami_name]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

// variables calculated after ami data is pulled
locals {
  // upstream platforms: build map from upstream module outputs. The
  // upstream modules/platform submodule declares windows_password but never
  // uses it -- it does not generate any user_data, so windows_2019/2022
  // (sourced from here) boot with no WinRM HTTPS listener and no firewall
  // rule for port 5986, unlike windows_2025 below. Apply the same
  // userdata_windows.tpl here for any winrm-connection platform so every
  // Windows worker actually configures WinRM over HTTPS on 5986.
  //
  // This is the platform's fixed/required default -- for winrm platforms it
  // is not optional (without it launchpad cannot connect at all), so it is
  // always present here. provision.tf composes it with any caller-supplied
  // nodegroup user_data (see nodegroups_wplatform) rather than this file
  // deciding precedence.
  upstream_platforms_with_ami = {
    for k, p in local.upstream_platform_keys : p => merge(module.platform[k].platform, {
      user_data = module.platform[k].platform.connection == "winrm" ? templatefile("${path.module}/userdata_windows.tpl", {
        windows_administrator_password = var.windows_password
      }) : ""
    })
  }

  // local platforms: build map matching the shape upstream module produces
  local_platforms_with_ami = {
    for p, def in local.lib_local_platform_definitions : p => merge(def, {
      ami              = data.aws_ami.local[p].id
      root_device_name = data.aws_ami.local[p].root_device_name
      user_data = def.connection == "winrm" ? templatefile("${path.module}/userdata_windows.tpl", {
        windows_administrator_password = var.windows_password
      }) : ""
    }) if contains(local.local_platform_keys, p)
  }

  // merge upstream + local into the single map consumed by provision.tf / launchpad.tf
  platforms_with_ami = merge(local.upstream_platforms_with_ami, local.local_platforms_with_ami)
}

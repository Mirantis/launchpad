// variables calculated before ami data is retrieved
locals {
  // find the unique platforms actually used in the node_group_definitions
  unique_used_platforms = distinct([for ngd in var.nodegroups : ngd.platform])

  // platforms defined in the upstream module
  upstream_platform_keys = [for p in local.unique_used_platforms : p if !contains(keys(local.lib_local_platform_definitions), p)]
  // platforms defined locally (not in upstream module)
  local_platform_keys = [for p in local.unique_used_platforms : p if contains(keys(local.lib_local_platform_definitions), p)]

  // local platform AMI definitions (supplements upstream module).
  // ubuntu_24.04 and ubuntu_26.04 used to live here; both are in the
  // upstream platform library as of provision-aws v0.3.0 and are resolved by
  // modules/platform now. ubuntu_22.04_fips has no upstream definition (the
  // library carries no FIPS images), so it stays local.
  lib_local_platform_definitions = {
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
  count   = length(local.upstream_platform_keys)
  source  = "terraform-mirantis-modules/provision-aws/mirantis//modules/platform"
  version = "0.3.0"

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
  // upstream platforms: build map from upstream module outputs. As of
  // provision-aws v0.3.0 modules/platform generates the required WinRM
  // user_data itself (Administrator password reset + WinRM-over-HTTPS
  // listener) from the windows_password passed above, so the output is used
  // as-is; launchpad no longer carries its own copy of that template.
  // provision.tf composes this platform default with any caller-supplied
  // nodegroup user_data (see nodegroups_wplatform).
  upstream_platforms_with_ami = {
    for k, p in local.upstream_platform_keys : p => module.platform[k].platform
  }

  // local platforms: build map matching the shape upstream module produces.
  // All local definitions are ssh platforms, so there is no fixed platform
  // user_data requirement -- provision.tf's caller-supplied nodegroup
  // user_data is used directly.
  local_platforms_with_ami = {
    for p, def in local.lib_local_platform_definitions : p => merge(def, {
      ami              = data.aws_ami.local[p].id
      root_device_name = data.aws_ami.local[p].root_device_name
      user_data        = ""
    }) if contains(local.local_platform_keys, p)
  }

  // merge upstream + local into the single map consumed by provision.tf / launchpad.tf
  platforms_with_ami = merge(local.upstream_platforms_with_ami, local.local_platforms_with_ami)
}

// variables calculated before ami data is retrieved
locals {
  // find the unique platforms actually used in the node_group_definitions
  unique_used_platforms = distinct([for ngd in var.nodegroups : ngd.platform])

  // platforms defined in the upstream module
  upstream_platform_keys = [for p in local.unique_used_platforms : p if !contains(keys(local.lib_local_platform_definitions), p)]
  // platforms defined locally (not in upstream module)
  local_platform_keys    = [for p in local.unique_used_platforms : p if contains(keys(local.lib_local_platform_definitions), p)]

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
    "windows_2025" = {
      ami_name       = "Windows_Server-2025-English-Core-Base-*"
      owner          = "801119661308"
      interface      = "Ethernet 3"
      connection     = "winrm"
      winrm_user     = "Administrator"
      winrm_useHTTPS = true
      winrm_insecure = true
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
  // upstream platforms: build map from upstream module outputs
  upstream_platforms_with_ami = {
    for k, p in local.upstream_platform_keys : p => module.platform[k].platform
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

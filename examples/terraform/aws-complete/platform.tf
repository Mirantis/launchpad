
// variables calculated before ami data is retrieved
locals {
  // find the unique platforms actually used in the node_group_definitions, so that we can combine platform definiton and ami data together
  // - this is unique to avoid repeated ami pulls for the same definition
  // - only node-group platforms are pulled to avoid pulling images data sources that are not used anywhere
  unique_used_platforms = distinct([for ngd in var.nodegroups : ngd.platform])

  // var.windows_password is null on Linux-only stacks, where no Windows node
  // exists to carry it; the consumers below take a string.  Its validation
  // rejects null when a windows_ platform is in use, so reaching here with
  // null means no Windows node is being provisioned.
  windows_password = var.windows_password == null ? "" : var.windows_password
}

module "platform" {
  count   = length(local.unique_used_platforms)
  source  = "terraform-mirantis-modules/provision-aws/mirantis//modules/platform"
  version = "0.1.8"

  platform_key     = local.unique_used_platforms[count.index]
  windows_password = local.windows_password
}

// variables calculated after ami data is pulled
locals {
  // convert platform ami list to a map
  platforms_with_ami = { for k, p in local.unique_used_platforms : p => module.platform[k].platform }
}

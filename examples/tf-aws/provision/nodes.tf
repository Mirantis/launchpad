
module "nodegroups" {
  for_each = var.nodegroups

  source = "./modules/nodegroup"

  name = "${var.name}-${each.key}"

  ami              = each.value.ami
  type             = each.value.type
  node_count       = each.value.count
  root_device_name = each.value.root_device_name
  volume_size      = each.value.volume_size
  user_data        = each.value.user_data
  key_pair         = each.value.keypair_id

  subnets         = module.vpc.public_subnets                                                                # TODO: right how we only support public nodes :(
  security_groups = [for k, sg in local.securitygroups_with_sg : sg.id if contains(sg.nodegroups, each.key)] # attach any sgs listed for this nodegroup

  tags = merge({
    stack = var.name
    },
    var.tags
  )
}

// locals created after node groups are provisioned.
locals {
  // combine node-group asg & node information after creation
  nodegroups = { for k, ng in var.nodegroups : k => merge(ng, {
    nodes : module.nodegroups[k].nodes
  }) }

  // a safer nodegroup listing that doesn't have any sensitive data.
  nodegroups_safer = { for k, ng in var.nodegroups : k => merge(ng, {
    nodes : [for j, i in module.nodegroups[k].nodes : {
      nodegroup       = k
      index           = j
      id              = "${k}-${j}"
      label           = "${var.name}-${k}-${j}"
      instance_id     = i.instance_id
      private_ip      = i.private_ip
      private_dns     = i.private_dns
      private_address = trimspace(coalesce(i.private_dns, i.private_ip, " "))
      public_ip       = i.public_ip
      public_dns      = i.public_dns
      public_address  = trimspace(coalesce(i.public_dns, i.public_ip, " "))
    }]
  }) }
}


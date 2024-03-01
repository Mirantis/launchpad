
// calculated before generating albs
locals {
  // combine the list of nodes for the ingress so that we can assign them as targets
  ingresses_withnodes = { for k, i in var.ingresses : k => merge(
    i,
    {
      nodes : concat([], [for k, n in local.nodegroups_safer : n.nodes if contains(i.nodegroups, k)]...),
    }
  ) }
}

module "ingress" {
  for_each = local.ingresses_withnodes
  source   = "./modules/ingress"

  name  = each.key
  stack = var.name

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.public_subnets

  instance_ids = [for n in each.value.nodes : n.instance_id]

  routes = each.value.routes

  tags = merge({
    stack = var.name
    },
    var.tags
  )
}

// calculated after lb is created
locals {
  // Add the lb for the lb to the ingress
  ingresses_withlb = { for k, i in var.ingresses : k => merge(i, module.ingress[k], { "lb_dns" : module.ingress[k].lb.dns_name }) }
}

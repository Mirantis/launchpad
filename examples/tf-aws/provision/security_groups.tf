
resource "aws_security_group" "sg" {
  for_each = var.securitygroups

  name        = "${var.name}-${each.key}"
  description = each.value.description
  vpc_id      = module.vpc.vpc_id

  tags = merge({
    stack = var.name
    role  = "sg"
    unit  = each.key
  }, var.tags)


  dynamic "ingress" {
    for_each = each.value.ingress_ipv4

    content {
      from_port   = ingress.value.from_port
      to_port     = ingress.value.to_port
      protocol    = ingress.value.protocol
      self        = ingress.value.self
      cidr_blocks = ingress.value.cidr_blocks
    }
  }

  dynamic "egress" {
    for_each = each.value.egress_ipv4

    content {
      from_port   = egress.value.from_port
      to_port     = egress.value.to_port
      protocol    = egress.value.protocol
      self        = egress.value.self
      cidr_blocks = egress.value.cidr_blocks
    }
  }
}

# locals generated after SGs are created
locals {
  securitygroups_with_sg = { for k, sg in var.securitygroups : k => merge(sg, aws_security_group.sg[k]) }
}


resource "aws_lb" "this" {
  name               = "${var.stack}-${var.name}-lb"
  internal           = false
  load_balancer_type = "network"

  subnets = var.subnet_ids

  tags = merge({
    ingress = var.name
    },
    var.tags
  )
}

resource "aws_lb_target_group" "targets" {
  for_each = var.routes

  name = "${var.stack}-${var.name}-${each.key}"

  vpc_id   = var.vpc_id
  port     = each.value.port_target
  protocol = each.value.protocol

  tags = merge({
    ingress = var.name
    target  = each.key
    },
    var.tags
  )
}

resource "aws_lb_listener" "listeners" {
  for_each = var.routes

  load_balancer_arn = aws_lb.this.arn
  port              = each.value.port_incoming
  protocol          = each.value.protocol

  default_action {
    target_group_arn = aws_lb_target_group.targets[each.key].arn
    type             = "forward"
  }
}

// === build a list of all target group attachments across all routes ===

locals {

  target_instances = concat([for k, t in var.routes : [for i in var.instance_ids : {
    target_group_arn = aws_lb_target_group.targets[k].arn,
    instance_id      = i,
    port             = t.port_target
  }]]...)

}

resource "aws_lb_target_group_attachment" "target_instances" {
  count = length(local.target_instances)

  target_group_arn = local.target_instances[count.index].target_group_arn
  target_id        = local.target_instances[count.index].instance_id
  port             = local.target_instances[count.index].port
}

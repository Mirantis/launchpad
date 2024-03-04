data "aws_instances" "nodes" {
  depends_on = [module.mg]

  instance_tags = {
    "aws:autoscaling:groupName" = module.mg.autoscaling_group_id
  }

  instance_state_names = ["pending", "running"]
}

data "aws_instance" "node" {
  depends_on = [module.mg, data.aws_instances.nodes]

  count = var.node_count // this is actually risky, but TF won't let us count the number of IDs in the instances resource

  instance_id = data.aws_instances.nodes.ids[count.index]
}


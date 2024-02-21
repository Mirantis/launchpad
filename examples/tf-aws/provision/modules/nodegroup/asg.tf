
module "mg" {
  source = "terraform-aws-modules/autoscaling/aws"

  # Autoscaling group
  name = var.name

  min_size         = var.node_count
  max_size         = var.node_count
  desired_capacity = var.node_count

  health_check_type = "EC2"

  vpc_zone_identifier = var.subnets
  security_groups     = var.security_groups

  image_id          = var.ami
  instance_type     = var.type
  ebs_optimized     = true
  enable_monitoring = true

  user_data = base64encode(var.user_data)
  key_name  = var.key_pair

  launch_template_name = var.name

  instance_refresh = {
    strategy = "Rolling"
    preferences = {
      checkpoint_delay       = 600
      checkpoint_percentages = [35, 70, 100]
      instance_warmup        = 300
      min_healthy_percentage = 75
      auto_rollback          = true
    }
    triggers = ["tag"]
  }

  network_interfaces = [
    {
      device_index                = 0
      delete_on_termination       = true
      associate_public_ip_address = true
    }
  ]

  block_device_mappings = [
    {
      # Root volume
      device_name = var.root_device_name
      no_device   = 0
      ebs = {
        delete_on_termination = true
        encrypted             = false
        volume_size           = var.volume_size
        volume_type           = "gp3"
      }
    },
  ]

  tags = merge({
    role = "mg"
    node = var.name
  }, var.tags)

  tag_specifications = [
    {
      resource_type = "instance"
      tags = {
        role = "mg-node"
        node = var.name
      }
    },
    {
      resource_type = "volume"
      tags = {
        role = "mg-volume"
        node = var.name
      }
    }
  ]
}

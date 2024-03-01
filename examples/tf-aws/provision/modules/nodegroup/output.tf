
output "asg" {
  value = module.mg
}

output "nodes" {
  value = data.aws_instance.node
}

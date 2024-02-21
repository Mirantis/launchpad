
output "vpc" {
  description = "VPC generated id"
  value       = module.vpc
}

output "nodegroups" {
  description = "Non-sensitive node group with generated node lists."
  value       = local.nodegroups_safer
}

output "ingresses" {
  description = "Created ingress data including urls"
  value       = local.ingresses_withlb
}

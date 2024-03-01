
output "platform" {
  description = "Image ami data for the platform"
  value       = local.platform_with_ami
  sensitive   = true // may have windows password in it
}

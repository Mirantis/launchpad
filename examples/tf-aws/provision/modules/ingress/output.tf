
output "lb" {
  description = "DNS entry for the ingress"
  value       = aws_lb.this
}

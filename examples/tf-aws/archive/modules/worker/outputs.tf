output "private_ips" {
    value = aws_instance.ucp_worker.*.private_ip
}
output "machines" {
  value = aws_instance.ucp_worker.*
}

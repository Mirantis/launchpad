output "lb_dns_name" {
    value = aws_lb.ucp_master.dns_name
}

output "public_ips" {
    value = aws_instance.ucp_master.*.public_ip
}

output "private_ips" {
    value = aws_instance.ucp_master.*.private_ip
}

output "machines" {
  value = aws_instance.ucp_master
}

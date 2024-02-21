output "keypair_id" {
  description = "AWS EC2 key-pair id"
  value       = aws_key_pair.keypair.id
}

output "private_key" {
  description = "Private key contents"
  value       = tls_private_key.ed25519.private_key_openssh
}

output "name" {
  description = "Name of the key"
  value       = aws_key_pair.keypair.key_name
}

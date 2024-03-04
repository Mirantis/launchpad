
resource "tls_private_key" "ed25519" {
  algorithm = "ED25519"
}

resource "aws_key_pair" "keypair" {
  key_name   = var.name
  public_key = tls_private_key.ed25519.public_key_openssh

  tags = merge({
    stack     = var.name
    algorythm = "ED25519"
    role      = "sshkey"
  }, var.tags)
}

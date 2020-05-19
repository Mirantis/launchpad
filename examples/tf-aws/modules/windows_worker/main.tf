resource "aws_security_group" "worker" {
  name        = "${var.cluster_name}-win-workers"
  description = "ucp cluster windows workers"
  vpc_id      = var.vpc_id
}

locals {
  subnet_count = length(var.subnet_ids)
}


resource "aws_instance" "ucp_worker" {
  count = var.worker_count

  tags = map(
    "Name", "${var.cluster_name}-win-worker-${count.index + 1}",
    "Role", "worker",
    "${var.kube_cluster_tag}", "shared"
  )

  instance_type          = var.worker_type
  iam_instance_profile   = var.instance_profile_name
  ami                    = var.image_id
  key_name               = var.ssh_key
  vpc_security_group_ids = [var.security_group_id, aws_security_group.worker.id]
  subnet_id              = var.subnet_ids[count.index % local.subnet_count]
  ebs_optimized          = true
#   user_data              = <<EOF
# <powershell>
# # Rename computer
# Rename-Computer -NewName (Resolve-DnsName -Name (Get-NetIPAddress -AddressFamily IPv4 -InterfaceAlias "Ethernet 3").IPAddress -DnsOnly).NameHost

# # Restart computer
# Restart-Computer
# </powershell>
# EOF


  lifecycle {
    ignore_changes = [ami]
  }

  root_block_device {
    volume_type = "gp2"
    volume_size = var.worker_volume_size
  }
}

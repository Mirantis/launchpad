resource "aws_security_group" "worker" {
  name        = "${var.cluster_name}-workers"
  description = "mke cluster workers"
  vpc_id      = var.vpc_id
}

locals {
  subnet_count = length(var.subnet_ids)
}


resource "aws_instance" "mke_worker" {
  count = var.worker_count

  tags = tomap({
    "Name" = "${var.cluster_name}-worker-${count.index + 1}",
    "Role" = "worker",
    (var.kube_cluster_tag) = "shared"
  })

  instance_type          = var.worker_type
  iam_instance_profile   = var.instance_profile_name
  ami                    = var.image_id
  key_name               = var.ssh_key
  vpc_security_group_ids = [var.security_group_id, aws_security_group.worker.id]
  subnet_id              = var.subnet_ids[count.index % local.subnet_count]
  ebs_optimized          = true
  user_data              = <<EOF
#!/bin/bash
# Use full qualified private DNS name for the host name.  Kube wants it this way.
HOSTNAME=$(curl http://169.254.169.254/latest/meta-data/hostname)
echo $HOSTNAME > /etc/hostname
sed -i "s|\(127\.0\..\.. *\)localhost|\1$HOSTNAME|" /etc/hosts
hostname $HOSTNAME
EOF

  lifecycle {
    ignore_changes = [ami]
  }

  root_block_device {
    volume_type = "gp2"
    volume_size = var.worker_volume_size
  }
}

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
  user_data              = <<EOF
<powershell>
# Install OpenSSH
Add-WindowsCapability -Online -Name OpenSSH.Client~~~~0.0.1.0
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
Start-Service sshd
Set-Service -Name sshd -StartupType 'Automatic'

# Configure ssh key authentication
mkdir c:\Users\Administrator\.ssh\
Invoke-WebRequest http://169.254.169.254/2012-01-12/meta-data/public-keys/0/openssh-key -UseBasicParsing -OutFile c:\Users\Administrator\.ssh\authorized_keys
Repair-AuthorizedKeyPermission C:\users\Administrator\.ssh\authorized_keys
Icacls C:\users\Administrator\.ssh\authorized_keys /remove “NT SERVICE\sshd”

$sshdConf = 'c:\ProgramData\ssh\sshd_config'
(Get-Content $sshdConf).replace('#PubkeyAuthentication yes', 'PubkeyAuthentication yes') | Set-Content $sshdConf
(Get-Content $sshdConf).replace('Match Group administrators', '#Match Group administrators') | Set-Content $sshdConf
(Get-Content $sshdConf).replace('       AuthorizedKeysFile __PROGRAMDATA__/ssh/administrators_authorized_keys', '#       AuthorizedKeysFile __PROGRAMDATA__/ssh/administrators_authorized_keys') | Set-Content $sshdConf
restart-service sshd
</powershell>
EOF


  lifecycle {
    ignore_changes = [ami]
  }

  root_block_device {
    volume_type = "gp2"
    volume_size = var.worker_volume_size
  }
}

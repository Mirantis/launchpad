resource "aws_security_group" "worker" {
  name        = "${var.cluster_name}-win-workers"
  description = "mke cluster windows workers"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = 5985
    to_port     = 5986
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

locals {
  subnet_count = length(var.subnet_ids)
}

resource "aws_instance" "mke_worker" {
  count = var.worker_count

  tags = tomap({
    "Name" = "${var.cluster_name}-win-worker-${count.index + 1}",
    "Role" = "worker",
    (var.kube_cluster_tag) = "shared"
  })

  instance_type          = var.worker_type
  iam_instance_profile   = var.instance_profile_name
  ami                    = var.image_id
  vpc_security_group_ids = [var.security_group_id, aws_security_group.worker.id]
  subnet_id              = var.subnet_ids[count.index % local.subnet_count]
  ebs_optimized          = true
  user_data              = <<EOF
<powershell>
$admin = [adsi]("WinNT://./administrator, user")
$admin.psbase.invoke("SetPassword", "${var.windows_administrator_password}")

# Snippet to enable WinRM over HTTPS with a self-signed certificate
# from https://gist.github.com/TechIsCool/d65017b8427cfa49d579a6d7b6e03c93
Write-Output "Disabling WinRM over HTTP..."
Disable-NetFirewallRule -Name "WINRM-HTTP-In-TCP"
Disable-NetFirewallRule -Name "WINRM-HTTP-In-TCP-PUBLIC"
Get-ChildItem WSMan:\Localhost\listener | Remove-Item -Recurse

Write-Output "Configuring WinRM for HTTPS..."
Set-Item -Path WSMan:\LocalHost\MaxTimeoutms -Value '1800000'
Set-Item -Path WSMan:\LocalHost\Shell\MaxMemoryPerShellMB -Value '1024'
Set-Item -Path WSMan:\LocalHost\Service\AllowUnencrypted -Value 'false'
Set-Item -Path WSMan:\LocalHost\Service\Auth\Basic -Value 'true'
Set-Item -Path WSMan:\LocalHost\Service\Auth\CredSSP -Value 'true'

New-NetFirewallRule -Name "WINRM-HTTPS-In-TCP" `
    -DisplayName "Windows Remote Management (HTTPS-In)" `
    -Description "Inbound rule for Windows Remote Management via WS-Management. [TCP 5986]" `
    -Group "Windows Remote Management" `
    -Program "System" `
    -Protocol TCP `
    -LocalPort "5986" `
    -Action Allow `
    -Profile Domain,Private

New-NetFirewallRule -Name "WINRM-HTTPS-In-TCP-PUBLIC" `
    -DisplayName "Windows Remote Management (HTTPS-In)" `
    -Description "Inbound rule for Windows Remote Management via WS-Management. [TCP 5986]" `
    -Group "Windows Remote Management" `
    -Program "System" `
    -Protocol TCP `
    -LocalPort "5986" `
    -Action Allow `
    -Profile Public

$Hostname = [System.Net.Dns]::GetHostByName((hostname)).HostName.ToUpper()
$pfx = New-SelfSignedCertificate -CertstoreLocation Cert:\LocalMachine\My -DnsName $Hostname
$certThumbprint = $pfx.Thumbprint
$certSubjectName = $pfx.SubjectName.Name.TrimStart("CN = ").Trim()

New-Item -Path WSMan:\LocalHost\Listener -Address * -Transport HTTPS -Hostname $certSubjectName -CertificateThumbPrint $certThumbprint -Port "5986" -force

Write-Output "Restarting WinRM Service..."
Stop-Service WinRM
Set-Service WinRM -StartupType "Automatic"
Start-Service WinRM
</powershell>
EOF


  lifecycle {
    ignore_changes = [ami]
  }

  root_block_device {
    volume_type = "gp2"
    volume_size = var.worker_volume_size
  }

  connection {
    type = "winrm"
    user = "Administrator"
    password = var.administrator_password
    timeout = "10m"
    https = "true"
    insecure = "true"
    port=5986
  }
}

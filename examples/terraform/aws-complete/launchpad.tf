// constants
locals {

  // role for MSR machines, so that we can detect if msr config is needed
  launchpad_role_msr = "msr"
  // only hosts with these roles will be used for launchpad_yaml
  launchpad_roles = ["manager", "worker", local.launchpad_role_msr]

}

// Launchpad configuration
variable "launchpad" {
  description = "launchpad install configuration"
  type = object({
    drain = bool

    mcr_version = string
    mke_version = string
    msr_version = string // unused if you have no MSR hosts

    mke_connect = object({
      username = string
      password = string
      insecure = bool // true if this endpoint will not use a valid certificate
    })

    skip_create  = bool
    skip_destroy = bool
  })
}

// locals calculated before the provision run
locals {
  // standard MKE ingresses
  launchpad_ingresses = {
    "mke" = {
      description = "MKE ingress for UI and Kube"
      nodegroups  = [for k, ng in var.nodegroups : k if ng.role == "manager"]
      tags        = local.tags
      routes = {
        "mke" = {
          port_incoming = 443
          port_target   = 443
          protocol      = "TCP"
        }
        "kube" = {
          port_incoming = 6443
          port_target   = 6443
          protocol      = "TCP"
        }
      }
    }
  }

  // standard MCR/MKE/MSR firewall rules [here we just leave it open until we can figure this out]
  launchpad_securitygroups = {
    "manager" = {
      description = "Common security group for manager nodes"
      nodegroups  = [for n, ng in var.nodegroups : n if ng.role == "manager"]
      tags        = local.common_tags
      ingress_ipv4 = [
        {
          description : "Allow https traffic from anywhere"
          from_port : 443
          to_port : 443
          protocol : "tcp"
          self : false
          cidr_blocks : ["0.0.0.0/0"]
        },
        {
          description : "Allow https traffic from anywhere for kube api server"
          from_port : 6443
          to_port : 6443
          protocol : "tcp"
          self : false
          cidr_blocks : ["0.0.0.0/0"]
        },
      ]
    }

  }

}

// prepare values to make it easier to feed into launchpad
locals {
  // The SAN URL for the MKE load balancer ingress that is for the MKE load balancer
  MKE_URL = module.provision.ingresses["mke"].lb_dns

  // flatten nodegroups into a set of objects with the info needed for each node, by combining the group details with the node detains
  launchpad_hosts_ssh = merge([for k, ng in local.nodegroups : { for l, ngn in ng.nodes : ngn.label => {
    label : ngn.label
    role : ng.role

    address : ngn.public_address

    ssh_address : ngn.public_ip
    ssh_user : ng.ssh_user
    ssh_port : ng.ssh_port
    ssh_key_path : abspath(local_sensitive_file.ssh_private_key.filename)
  } if contains(local.launchpad_roles, ng.role) && ng.connection == "ssh" }]...)
  launchpad_hosts_winrm = merge([for k, ng in local.nodegroups : { for l, ngn in ng.nodes : ngn.label => {
    label : ngn.label
    role : ng.role

    address : ngn.public_address

    winrm_address : ngn.public_ip
    winrm_user : ng.winrm_user
    winrm_password : var.windows_password
    winrm_useHTTPS : ng.winrm_useHTTPS
    winrm_insecure : ng.winrm_insecure
  } if contains(local.launchpad_roles, ng.role) && ng.connection == "winrm" }]...)

  // decide if we need msr configuration (the [0] is needed to prevent an error of no msr instances exit)
  has_msr = sum(concat([0], [for k, ng in local.nodegroups : ng.count if ng.role == local.launchpad_role_msr])) > 0
}

resource "launchpad_config" "cluster" {
  skip_create  = var.launchpad.skip_create
  skip_destroy = var.launchpad.skip_destroy

  metadata {
    name = var.name
  }
  spec {
    mcr {
      version = var.launchpad.mcr_version
    }
    mke {
      version        = var.launchpad.mke_version
      admin_password = var.launchpad.mke_connect.password
      install_flags = [
        "--san=${local.MKE_URL}",
        "--cloud-provider=external"
      ]
    }

    dynamic "msr" {
      for_each = local.has_msr ? [1] : []

      content {
        version    = var.launchpad.msr_version
        image_repo = "docker.io/mirantis"

        install_flags = [
          "--ucp-insecure-tls"
        ]
      }
    }

    // add hosts for every *nix/ssh host 
    dynamic "host" {
      for_each = nonsensitive(local.launchpad_hosts_ssh)

      content {
        role = host.value.role
        ssh {
          address  = host.value.ssh_address
          key_path = host.value.ssh_key_path
          user     = host.value.ssh_user
        }
      }
    }

    // add hosts for every windows/winrm host
    dynamic "host" {
      for_each = nonsensitive(local.launchpad_hosts_winrm)

      content {
        role = host.value.role
        winrm {
          address   = host.value.winrm_address
          user      = host.value.winrm_user
          password  = host.value.winrm_password
          insecure  = host.value.winrm_insecure
          use_https = host.value.winrm_useHTTPS
        }
      }
    }
  }
}

// ------- Ye old launchpad yaml (just for debugging)
output "launchpad_yaml" {
  description = "launchpad config file yaml (for debugging)"
  sensitive   = true
  value       = <<-EOT
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke%{if local.has_msr}+msr%{endif}
metadata:
  name: ${var.name}
spec:
  cluster:
    prune: false
  hosts:
%{~for h in local.launchpad_hosts_ssh}
  # ${h.label} (ssh)
  - role: ${h.role}
    ssh:
      address: ${h.ssh_address}
      user: ${h.ssh_user}
      keyPath: ${h.ssh_key_path}
%{~endfor}
%{~for h in local.launchpad_hosts_winrm}
  # ${h.label} (winrm)
  - role: ${h.role}
    winRM:
      address: ${h.winrm_address}
      user: ${h.winrm_user}
      password: ${h.winrm_password}
      useHTTPS: ${h.winrm_useHTTPS}
      insecure: ${h.winrm_insecure}
%{~endfor}
  mke:
    version: ${var.launchpad.mke_version}
    imageRepo: docker.io/mirantis
    adminUsername: ${var.launchpad.mke_connect.username}
    adminPassword: ${var.launchpad.mke_connect.password}
    installFlags: 
    - "--san=${local.MKE_URL}"
    - "--nodeport-range=32768-35535"
    upgradeFlags:
    - "--force-recent-backup"
    - "--force-minimums"
  mcr:
    version: ${var.launchpad.mcr_version}
    repoURL: https://repos.mirantis.com
    installURLLinux: https://get.mirantis.com/
    installURLWindows: https://get.mirantis.com/install.ps1
    channel: stable
%{if local.has_msr}
  msr:
    version: ${var.launchpad.msr_version}
    imageRepo: docker.io/mirantis
%{endif}
EOT

}

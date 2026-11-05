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

    mcr_channel  = string
    mcr_repo_url = optional(string, "https://repos.mirantis.com")
    mke_version  = string
    msr_version  = string // unused if you have no MSR hosts

    mke_connect = object({
      username = string
      password = string
      insecure = bool // true if this endpoint will not use a valid certificate
    })
  })
}

// locals calculated before the provision run
locals {
  // standard MKE ingresses
  launchpad_ingresses = {
    "mke" = {
      description = "MKE ingress for UI and Kube"
      nodegroups  = [for k, ng in var.nodegroups : k if ng.role == "manager"]

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
    "msr" = {
      description = "MSR ingress for UI and docker"
      nodegroups  = [for k, ng in var.nodegroups : k if ng.role == "msr"]

      routes = {
        "msr" = {
          port_incoming = 443
          port_target   = 443
          protocol      = "TCP"
        }
      }
    }
  }

  // MCR/MKE/MSR firewall rules.
  //
  // Node-to-node traffic inside the group stays unrestricted: MKE, swarm and
  // the kubelet use a wide and version-dependent port range between cluster
  // members, and every member is in this security group.
  //
  // Everything arriving from outside the group is restricted to the ports the
  // cluster actually serves, from the CIDRs the operator names in
  // var.admin_cidrs.  The two load-balanced ports additionally accept the VPC
  // CIDR, because the MKE and MSR load balancers defined in
  // local.launchpad_ingresses above reach the nodes from inside the VPC
  // rather than from an administrator's address, and their health checks
  // would fail if only var.admin_cidrs were allowed.
  launchpad_securitygroups = {
    "permissive" = {
      description = "Common SG for all cluster machines"
      nodegroups  = [for n, ng in var.nodegroups : n]
      ingress_ipv4 = [
        {
          description : "Cluster-internal traffic between members of this group"
          from_port : 0
          to_port : 0
          protocol : "-1"
          self : true
          cidr_blocks : []
        },
        {
          description : "SSH for launchpad host configuration"
          from_port : 22
          to_port : 22
          protocol : "tcp"
          self : false
          cidr_blocks : var.admin_cidrs
        },
        {
          description : "WinRM over HTTPS for launchpad host configuration"
          from_port : 5986
          to_port : 5986
          protocol : "tcp"
          self : false
          cidr_blocks : var.admin_cidrs
        },
        {
          description : "MKE and MSR web/API, from administrators and from the load balancers"
          from_port : 443
          to_port : 443
          protocol : "tcp"
          self : false
          cidr_blocks : concat(var.admin_cidrs, [var.network.cidr])
        },
        {
          description : "Kubernetes API, from administrators and from the load balancer"
          from_port : 6443
          to_port : 6443
          protocol : "tcp"
          self : false
          cidr_blocks : concat(var.admin_cidrs, [var.network.cidr])
        }
      ]
      egress_ipv4 = [
        {
          description : "Permissive outgoing traffic"
          from_port : 0
          to_port : 0
          protocol : "-1"
          cidr_blocks : ["0.0.0.0/0"]
          self : false
        }
      ]
    }
  }

}

// prepare values to make it easier to feed into launchpad
locals {
  // The SAN URL for the MKE load balancer ingress that is for the MKE load balancer
  MKE_URL = module.provision.ingresses["mke"].lb_dns
  MSR_URL = module.provision.ingresses["msr"].lb_dns

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
    winrm_password : local.windows_password
    winrm_useHTTPS : ng.winrm_useHTTPS
    winrm_insecure : ng.winrm_insecure
  } if contains(local.launchpad_roles, ng.role) && ng.connection == "winrm" }]...)

  // decide if we need msr configuration (the [0] is needed to prevent an error of no msr instances exit)
  has_msr = sum(concat([0], [for k, ng in local.nodegroups : ng.count if ng.role == local.launchpad_role_msr])) > 0
}

// ------- Ye old launchpad yaml (just for debugging)
//
// The credentials are referenced by environment variable rather than written
// into the document.  launchpad expands these through envsubst when it loads
// the config (pkg/config), and docs/usage/getting-started.md tells users to
// source credentials from the environment; a generated file that inlines them
// contradicts that and leaves the password in every copy of the YAML, in
// terraform state, and in anything that prints either.  The consumer exports
// LAUNCHPAD_MKE_ADMIN_PASSWORD and, for Windows stacks,
// LAUNCHPAD_WINRM_PASSWORD before running launchpad; an unset variable fails
// config loading loudly rather than silently installing with an empty
// password (PRODENG-3751).
locals {
  launchpad_yaml = <<-EOT
apiVersion: launchpad.mirantis.com/mke/v1.6
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
      password: $${LAUNCHPAD_WINRM_PASSWORD}
      useHTTPS: ${h.winrm_useHTTPS}
      insecure: ${h.winrm_insecure}
%{~endfor}
  mcr:
    channel: ${var.launchpad.mcr_channel}
    repoURL: ${var.launchpad.mcr_repo_url}
    installURLWindows: https://get.mirantis.com/install.ps1
    prune: true
  mke:
    version: ${var.launchpad.mke_version}
    imageRepo: docker.io/mirantis
    adminUsername: ${var.launchpad.mke_connect.username}
    adminPassword: $${LAUNCHPAD_MKE_ADMIN_PASSWORD}
    installFlags: 
    - "--san=${local.MKE_URL}"
    - "--default-node-orchestrator=kubernetes"
    - "--nodeport-range=32768-35535"
    upgradeFlags:
    - "--force-recent-backup"
    - "--force-minimums"
%{if local.has_msr}
  msr:
    version: ${var.launchpad.msr_version}
    imageRepo: docker.io/mirantis
    "replicaIDs": "sequential"
    installFlags:
    - "--ucp-insecure-tls"
    - "--dtr-external-url=${local.MSR_URL}"
%{endif}
EOT

}

output "launchpad_yaml" {
  description = "launchpad config file yaml (for debugging)"
  sensitive   = true
  value       = local.launchpad_yaml
}

output "mke_connect" {
  description = "Connection information for connecting to MKE"
  sensitive   = true
  value = {
    host     = local.MKE_URL
    username = var.launchpad.mke_connect.username
    password = var.launchpad.mke_connect.password
    insecure = var.launchpad.mke_connect.insecure
  }
}

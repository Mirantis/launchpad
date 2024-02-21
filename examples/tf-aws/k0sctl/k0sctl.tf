// constants
locals {

  // only hosts with these roles will be used for k0s
  k0s_roles = ["controller", "worker"]

}

variable "k0sctl" {
  description = "K0sctl install configuration"
  type = object({
    version = string

    no_wait  = bool
    no_drain = bool

    force = bool

    disable_downgrade_check = bool

    restore_from = string
  })
}

// locals calculated before the provision run
locals {
  // standard MKE ingresses
  k0s_ingresses = {
    "kube" = {
      description = "ingress for Kube API"
      nodegroups  = [for k, ng in var.nodegroups : k if ng.role == "controller"]

      routes = {
        "kube" = {
          port_incoming = 6443
          port_target   = 6443
          protocol      = "TCP"
        }
      }
    }
  }

  // standard firewall rules [here we just leave it open until we can figure this out]
  k0s_securitygroups = {
    "permissive" = {
      description = "Common permissive SG for all cluster machines"
      nodegroups  = [for n, ng in var.nodegroups : n]

      ingress_ipv4 = [
        {
          description : "Permissive internal traffic [BAD RULE]"
          from_port : 0
          to_port : 0
          protocol : "-1"
          self : true
          cidr_blocks : []
        },
        {
          description : "Permissive external traffic [BAD RULE]"
          from_port : 0
          to_port : 0
          protocol : "-1"
          self : false
          cidr_blocks : ["0.0.0.0/0"]
        },
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

  // This should likely be built using a template
  k0s_config = <<EOT
apiVersion: k0s.k0sproject.io/v1beta1
kind: ClusterConfig
metadata:
  name: k0s
spec:
  controllerManager: {}
  extensions:
    helm:
      concurrencyLevel: 5
      charts: null
      repositories: null
    storage:
      create_default_storage_class: false
      type: external_storage
  installConfig:
    users:
      etcdUser: etcd
      kineUser: kube-apiserver
      konnectivityUser: konnectivity-server
      kubeAPIserverUser: kube-apiserver
      kubeSchedulerUser: kube-scheduler
EOT


  // The SAN URL for the kubernetes load balancer ingress that is for the MKE load balancer
  KUBE_URL = module.provision.ingresses["kube"].lb_dns

  // flatten nodegroups into a set of objects with the info needed for each node, by combining the group details with the node detains
  hosts_ssh = tolist(concat([for k, ng in local.nodegroups : [for l, ngn in ng.nodes : {
    label : ngn.label
    role : ng.role

    address : ngn.public_address

    ssh_address : ngn.public_ip
    ssh_user : ng.ssh_user
    ssh_port : ng.ssh_port
    ssh_key_path : abspath(local_sensitive_file.ssh_private_key.filename)
  } if contains(local.k0s_roles, ng.role) && ng.connection == "ssh"]]...))

}

output "hosts_ssh" {
  value     = local.hosts_ssh
  sensitive = true
}


// launchpad install from provisioned cluster
resource "k0sctl_config" "cluster" {
  # Tell the k0s provider to not bother installing/uninstalling

  metadata {
    name = var.name
  }

  spec {
    // ssh hosts
    dynamic "host" {
      for_each = nonsensitive(local.hosts_ssh)

      content {
        role = host.value.role
        ssh {
          address  = host.value.ssh_address
          user     = host.value.ssh_user
          key_path = host.value.ssh_key_path
        }
      }
    }

    # K0s configuration
    k0s {
      version = var.k0sctl.version
      config  = local.k0s_config
    } // k0s

  } // spec
}

output "kube_connect" {
  description = "parametrized config for kubernetes/helm provider configuration"
  sensitive   = true
  value = {
    host               = k0sctl_config.cluster.kube_host
    client_certificate = k0sctl_config.cluster.client_cert
    client_key         = k0sctl_config.cluster.private_key
    ca_certificate     = k0sctl_config.cluster.ca_cert
    tlsverifydisable   = k0sctl_config.cluster.kube_skiptlsverify
  }
}

output "kube_yaml" {
  description = "kubernetes config file yaml (for debugging)"
  sensitive   = true
  value       = k0sctl_config.cluster.kube_yaml
}

output "k0sctl_yaml" {
  description = "k0sctl config file yaml (for debugging)"
  sensitive   = true
  value       = k0sctl_config.cluster.k0s_yaml
}

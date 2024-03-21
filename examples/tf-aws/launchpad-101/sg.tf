locals {
  common_security_groups = {
    "permissive" = {
      description = "Common SG for all cluster machines"
      nodegroups  = [for n, ng in var.nodegroups : n]
      tags        = merge(local.common_tags, local.kube_tags)
      ingress_ipv4 = [
        {
          description : "Permissive internal traffic [BAD RULE]"
          from_port : 0
          to_port : 0
          protocol : "-1"
          self : true
          cidr_blocks : []
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

    "ssh" = {
      description = "Security for group for openning ssh port"
      nodegroups  = [for n, ng in local.nodegroups_wplatform : n if ng.platform == ""] # platform attribute is empty for linux in aws_ami data source
      tags        = local.common_tags
      ingress_ipv4 = [
        {
          description : "Allow ssh traffic from anywhere"
          from_port : 22
          to_port : 22
          protocol : "tcp"
          self : false
          cidr_blocks : ["0.0.0.0/0"]
        },
      ]
    }

    "winrm" = {
      description = "Security for group for openning winrm ports"
      nodegroups  = [for n, ng in local.nodegroups_wplatform : n if ng.platform == "windows"]
      tags        = local.common_tags
      ingress_ipv4 = [
        {
          description : "Allow winrm traffic from anywhere"
          from_port : 5985
          to_port : 5986
          protocol : "tcp"
          self : false
          cidr_blocks : ["0.0.0.0/0"]
        },
      ]
    }
  }
}

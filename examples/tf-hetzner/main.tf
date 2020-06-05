variable "hcloud_token" {
    description = "Hetzner API token"
}

provider "hcloud" {
  token = "${var.hcloud_token}"
}

variable "ssh_keys" {
    default = []
}

variable "ssh_user" {
    default = "root"
}

variable "cluster_name" {
    default = "ucp"
}

variable "location" {
    default = "hel1"
}

variable "image" {
    default = "ubuntu-18.04"
}

variable "master_type" {
    default = "cx31"
}

variable "worker_count" {
    default = 1
}

variable "master_count" {
    default = 1
}

variable "worker_type" {
    default = "cx31"
}

resource "hcloud_server" "master" {
    count = "${var.master_count}"
    name = "${var.cluster_name}-master-${count.index}"
    image = "${var.image}"
    server_type = "${var.master_type}"
    ssh_keys = "${var.ssh_keys}"
    location = "${var.location}"
    labels = {
        role = "manager"
    }
}

resource "hcloud_server" "worker" {
    count = "${var.worker_count}"
    name = "${var.cluster_name}-worker-${count.index}"
    image = "${var.image}"
    server_type = "${var.worker_type}"
    ssh_keys = "${var.ssh_keys}"
    location = "${var.location}"
    labels = {
        role = "worker"
    }
}

output "ucp_cluster" {
    value = {
        apiVersion = "launchpad.mirantis.com/v1beta2"
        kind = "UCP"
        spec = {
            hosts = [
                for host in concat(hcloud_server.master, hcloud_server.worker) : {
                    address      = host.ipv4_address
                    user    = "root"
                    role    = host.labels.role
                }
            ]
        }
    }
}

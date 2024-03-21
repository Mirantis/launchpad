
provider "kubernetes" {
  host                   = resource.mke_clientbundle.admin.kube_host
  client_certificate     = resource.mke_clientbundle.admin.client_cert
  client_key             = resource.mke_clientbundle.admin.private_key
  cluster_ca_certificate = resource.mke_clientbundle.admin.ca_cert

  insecure = resource.mke_clientbundle.admin.kube_skiptlsverify
}

provider "helm" {
  kubernetes {
    host                   = resource.mke_clientbundle.admin.kube_host
    client_certificate     = resource.mke_clientbundle.admin.client_cert
    client_key             = resource.mke_clientbundle.admin.private_key
    cluster_ca_certificate = resource.mke_clientbundle.admin.ca_cert
  }
}

output "kubeconfig" {
  description = "the contents of the kubeconfig yaml file"
  value       = mke_clientbundle.admin.kube_yaml
  sensitive   = true
}

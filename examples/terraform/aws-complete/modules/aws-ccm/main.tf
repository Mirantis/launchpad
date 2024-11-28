resource "helm_release" "aws_ccm" {
  name       = "aws-cloud-controller-manager"
  repository = "https://kubernetes.github.io/cloud-provider-aws"
  chart      = "aws-cloud-controller-manager"
  namespace  = "kube-system"
  values = [
    file("${path.module}/values.yaml")
  ]
}

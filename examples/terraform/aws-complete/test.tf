# # Some testing resources to prove that dynamic volume provision and ALB provisioning are working

# resource "kubernetes_namespace" "test" {
#   metadata {
#     name = "test"
#   }

#   depends_on = [
#     mke_clientbundle.admin
#   ]
# }

# locals {
#   gp3_storage_class_name = "test-gp3"
#   gp3_test_pvc_name      = "test-pvc"
#   test_volume_name       = "test-volume"
# }

# resource "kubernetes_storage_class" "test_gp3" {
#   metadata {
#     name = local.gp3_storage_class_name
#   }
#   storage_provisioner = "ebs.csi.aws.com"
#   reclaim_policy      = "Delete"
#   parameters = {
#     type = "gp3"
#   }
# }


# resource "kubernetes_persistent_volume_claim" "ebs_gp3" {
#   metadata {
#     name      = local.gp3_test_pvc_name
#     namespace = kubernetes_namespace.test.metadata[0].name
#   }
#   spec {
#     access_modes       = ["ReadWriteOnce"]
#     storage_class_name = local.gp3_storage_class_name
#     resources {
#       requests = {
#         storage = "5Gi"
#       }
#     }
#   }
# }

# resource "kubernetes_deployment" "nginx" {
#   metadata {
#     name      = "nginx-deployment"
#     namespace = kubernetes_namespace.test.metadata[0].name
#   }

#   spec {
#     replicas = 1

#     selector {
#       match_labels = {
#         app = "nginx"
#       }
#     }

#     template {
#       metadata {
#         labels = {
#           app = "nginx"
#         }
#       }

#       spec {
#         container {
#           image = "nginx:latest"
#           name  = "nginx"

#           port {
#             container_port = 80
#           }

#           volume_mount {
#             name       = local.test_volume_name
#             mount_path = "/tmp"
#           }
#         }
#         volume {
#           name = local.test_volume_name
#           persistent_volume_claim {
#             claim_name = local.gp3_test_pvc_name
#           }
#         }
#       }
#     }
#   }
# }

# resource "kubernetes_service" "nginx" {
#   metadata {
#     name      = "nginx-service"
#     namespace = kubernetes_namespace.test.metadata[0].name
#   }

#   spec {
#     selector = {
#       app = "nginx"
#     }

#     port {
#       protocol    = "TCP"
#       port        = 80
#       target_port = 80
#     }

#     type = "LoadBalancer"
#   }
# }

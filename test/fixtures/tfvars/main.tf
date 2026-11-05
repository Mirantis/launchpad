# Test fixture for TestSecretVarsRoundTripThroughTerraform (test/terraform_test.go).
# This is NOT a usable example of a launchpad stack; see examples/terraform/ for
# those. It exists only to exercise how test.TerraformOptions passes variables:
# it declares no providers, creates no resources and reaches no network, so
# `terraform init` and `terraform apply` run offline and cost nothing.
#
# It mirrors the shape examples/terraform/aws-simple declares for the
# credential-bearing variables, so a change in how those variables are encoded
# fails here rather than in a smoke run nobody can execute locally.

terraform {
  required_version = ">= 1.0"
}

variable "name" {
  description = "Stack name; carries no credential, passed as -var like the real stacks."
  type        = string
}

variable "launchpad" {
  description = "Mirrors aws-simple's launchpad variable, including the nested MKE admin password."
  type = object({
    drain = bool

    mcr_channel  = string
    mcr_repo_url = optional(string, "https://repos.mirantis.com")
    mke_version  = string
    msr_version  = string

    mke_connect = object({
      username = string
      password = string
      insecure = bool
    })
  })
}

variable "windows_password" {
  description = "Mirrors aws-simple's windows_password variable."
  type        = string
  default     = ""
}

# Mirrors aws-simple's launchpad_yaml: a sensitive output whose value embeds
# the credentials, which is what the test asserts never reaches the log.
output "cluster_yaml" {
  description = "Rendered cluster config carrying both passwords."
  sensitive   = true
  value = yamlencode({
    apiVersion = "launchpad.mirantis.com/mke/v1.6"
    spec = {
      mke = {
        adminUsername = var.launchpad.mke_connect.username
        adminPassword = var.launchpad.mke_connect.password
        version       = var.launchpad.mke_version
      }
      hosts = [
        {
          winRM = {
            user     = "Administrator"
            password = var.windows_password
          }
        }
      ]
    }
  })
}

# Echoes the launchpad variable back so the test can assert the object survives
# the TF_VAR_* transport unchanged, nested password included.
output "launchpad_echo" {
  description = "The launchpad variable as terraform parsed it."
  sensitive   = true
  value       = var.launchpad
}

# Carries no credential: used to assert that ordinary logging still happens.
output "name_echo" {
  description = "The stack name as terraform parsed it."
  value       = var.name
}

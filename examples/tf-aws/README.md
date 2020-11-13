# Bootstrapping MKE cluster on AWS

This directory provides an example flow with Launchpad tool together with Terraform.

## Steps

1. Create terraform.tfvars file with needed details. You can use the provided terraform.tfvars.example as a baseline.
2. `terraform init`
3. `terraform apply`
4. `terraform output mke_cluster | launchpad apply`
5. Profit! :)


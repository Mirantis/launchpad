# Bootstrapping UCP cluster on Hetzner

This directory provides an example flow with mcc tool together with Terraform.

## Steps

1. Create terraform.tfvars file with needed details. You can use the provided terraform.tfvars.example as a baseline.
2. `terraform init`
3. `terraform apply`
4. `terraform output -json | yq r --prettyPrint - ucp_cluster.value > cluster.yaml `
5. `launchpad apply --config ./examples/tf-hetzner/cluster.yaml`
6. Profit! :)


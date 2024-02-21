
data "aws_availability_zones" "available" {
  state = "available"
}

locals {
  // try to get as many azs as needed to stripe subnets across them.
  az_subsets = slice(data.aws_availability_zones.available.names, 0, 1)
}

module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  name = var.name
  cidr = var.cidr

  azs             = local.az_subsets
  public_subnets  = [for i in range(var.public_subnet_count) : cidrsubnet(var.cidr, 4, i)]
  private_subnets = [for i in range(var.public_subnet_count, var.private_subnet_count + var.public_subnet_count) : cidrsubnet(var.cidr, 4, i)]

  enable_nat_gateway = var.enable_nat_gateway
  enable_vpn_gateway = var.enable_vpn_gateway

  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = merge({
    stack = var.name
    role  = "vpc"
    unit  = "primary"
  }, var.tags)
}

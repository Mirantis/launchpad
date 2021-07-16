data "aws_availability_zones" "all" {}

# Network VPC, gateway, and routes

resource "aws_vpc" "network" {
  cidr_block                       = var.host_cidr
  assign_generated_ipv6_cidr_block = true
  enable_dns_support               = true
  enable_dns_hostnames             = true

  tags = tomap({"Name" = var.cluster_name})
}

resource "aws_internet_gateway" "gateway" {
  vpc_id = aws_vpc.network.id

  tags = tomap({"Name" = var.cluster_name})
}

resource "aws_route_table" "default" {
  vpc_id = aws_vpc.network.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gateway.id
  }

  tags = tomap({"Name" = var.cluster_name})
}

locals {
  kube_cluster_tag = "kubernetes.io/cluster/${var.cluster_name}"
}


# Subnets (one per availability zone)
resource "aws_subnet" "public" {
  count = 2

  vpc_id            = aws_vpc.network.id
  availability_zone = data.aws_availability_zones.all.names[count.index]

  cidr_block                      = cidrsubnet(var.host_cidr, 8, count.index)
  map_public_ip_on_launch         = true

  tags = tomap({(local.kube_cluster_tag) = "true"})
}

resource "aws_route_table_association" "public" {
  count          = length(aws_subnet.public)
  route_table_id = aws_route_table.default.id
  subnet_id      = element(aws_subnet.public.*.id, count.index)
}

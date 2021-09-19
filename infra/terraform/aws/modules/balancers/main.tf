terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "~> 3.27"
    }
  }

  required_version = ">= 0.14.9"
}

provider "aws" {
  profile = "default"
  region = "us-east-2"
}

variable "stack" {
  type = string
  default = "perfa"
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

data "aws_vpc" "rkcy" {
  filter {
    name = "tag:Name"
    values = ["rkcy_${var.stack}_vpc"]
  }
}

variable "bastion_hosts" {
  type = list
  default = ["bastion-0.perfa.rkcy.net"]
}

variable "jaeger_query_hosts" {
  type = list
  default = ["jaeger-query-0.perfa.local.rkcy.net:16686"]
}

data "aws_route53_zone" "zone" {
  name = "rkcy.net"
}

data "aws_subnet" "rkcy_edge" {
  count = var.edge_count
  vpc_id = data.aws_vpc.rkcy.id

  filter {
    name = "tag:Name"
    values = ["rkcy_${var.stack}_edge_${count.index}_sn"]
  }
}

data "aws_subnet" "rkcy_app" {
  count = var.app_count
  vpc_id = data.aws_vpc.rkcy.id

  filter {
    name = "tag:Name"
    values = ["rkcy_${var.stack}_app_${count.index}_sn"]
  }
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

variable "edge_count" {
  type = number
  default = 1
}

variable "app_count" {
  type = number
  default = 1
}

module "nginx_edge" {
  source = "../../modules/nginx"

  stack = var.stack
  cluster = "edge"
  vpc = data.aws_vpc.rkcy
  subnet = data.aws_subnet.rkcy_edge
  dns_zone = data.aws_route53_zone.zone
  bastion_hosts = var.bastion_hosts
  inbound_cidr = "${chomp(data.http.myip.body)}/32"
  public = true
  nginx_count = var.edge_count
  routes = [
    {
      name = "jaeger",
      servers = var.jaeger_query_hosts,
    },
  ]
}

module "nginx_app" {
  source = "../../modules/nginx"

  stack = var.stack
  cluster = "app"
  vpc = data.aws_vpc.rkcy
  subnet = data.aws_subnet.rkcy_app
  dns_zone = data.aws_route53_zone.zone
  bastion_hosts = var.bastion_hosts
  inbound_cidr = data.aws_vpc.rkcy.cidr_block
  public = false
  nginx_count = var.app_count
  routes = []
}

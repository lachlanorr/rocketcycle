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
}

variable "dns_zone" {
  type = any
}

variable "vpc" {
  type = any
}

variable "subnet_edge" {
  type = any
}

variable "subnet_app" {
  type = any
}

variable "bastion_hosts" {
  type = list
}

variable "jaeger_query_hosts" {
  type = list
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
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
  vpc = var.vpc
  subnet = var.subnet_edge
  dns_zone = var.dns_zone
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
  vpc = var.vpc
  subnet = var.subnet_app
  dns_zone = var.dns_zone
  bastion_hosts = var.bastion_hosts
  inbound_cidr = var.vpc.cidr_block
  public = false
  nginx_count = var.app_count
  routes = []
}

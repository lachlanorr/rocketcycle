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

variable "bastion_ips" {
  type = list
}

variable "jaeger_query_hosts" {
  type = list
}
variable "jaeger_query_port" {
  type = number
}

variable "jaeger_collector_hosts" {
  type = list
}
variable "jaeger_collector_port" {
  type = number
}

variable "otelcol_hosts" {
  type = list
}
variable "otelcol_port" {
  type = number
}

variable "prometheus_hosts" {
  type = list
}
variable "prometheus_port" {
  type = number
}

variable "grafana_hosts" {
  type = list
}
variable "grafana_port" {
  type = number
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

variable "public" {
  type = bool
}

locals {
  default_routes = [
    {
      name = "grafana"
      hosts = var.grafana_hosts
      port = var.grafana_port
      grpc = false
    },
    {
      name = "jaeger"
      hosts = var.jaeger_query_hosts
      port = var.jaeger_query_port
      grpc = false
    },
  ]
  private_routes = [
    {
      name = "jaegercol"
      hosts = var.jaeger_collector_hosts
      port = var.jaeger_collector_port
      grpc = true
    },
    {
      name = "otelcol"
      hosts = var.otelcol_hosts
      port = var.otelcol_port
      grpc = true
    },
    {
      name = "prometheus"
      hosts = var.prometheus_hosts
      port = var.prometheus_port
      grpc = false
    },
  ]
  public_routes = concat(local.default_routes, var.public ? tolist(local.private_routes) : tolist([]))
}

module "nginx_edge" {
  source = "../../modules/nginx"

  stack = var.stack
  cluster = "edge"
  vpc = var.vpc
  subnet = var.subnet_edge
  dns_zone = var.dns_zone
  bastion_ips = var.bastion_ips
  inbound_cidr = "${chomp(data.http.myip.body)}/32"
  public = true
  nginx_count = var.edge_count
  routes = local.public_routes
}

module "nginx_app" {
  source = "../../modules/nginx"

  stack = var.stack
  cluster = "app"
  vpc = var.vpc
  subnet = var.subnet_app
  dns_zone = var.dns_zone
  bastion_ips = var.bastion_ips
  inbound_cidr = var.vpc.cidr_block
  public = var.public
  nginx_count = var.app_count
  routes = local.private_routes
}

output "balancer_internal_urls" {
  value = {
    edge = module.nginx_edge.balancer_internal_url
    app = module.nginx_app.balancer_internal_url
  }
}

output "balancer_external_urls" {
  value = {
    edge = module.nginx_edge.balancer_external_url
    app = module.nginx_app.balancer_external_url
  }
}

output "nginx_hosts" {
  value = {
    edge = sort(module.nginx_edge.nginx_hosts)
    app = sort(module.nginx_app.nginx_hosts)
  }
}

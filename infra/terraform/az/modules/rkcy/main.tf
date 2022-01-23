terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "=2.91.0"
    }
  }
}

provider "azurerm" {
  features {}
}

variable "image_resource_group_name" {
  type = string
}

variable "stack" {
  type = string
}

variable "dns_zone" {
  type = string
}

variable "cidr_block" {
  type = string
}

variable "public" {
  type = bool
}

module "network" {
  source = "../../modules/network"

  cidr_block = var.cidr_block
  image_resource_group_name = var.image_resource_group_name
  stack = var.stack
  dns_zone = var.dns_zone
}

#module "dev" {
#  source = "../../modules/dev"
#
#  stack = module.network.stack
#  dns_zone = module.network.dns_zone
#  resource_group = module.network.resource_group
#  subnet_edge = module.network.subnet_edge
#  postgresql_hosts = module.postgresql.postgresql_hosts
#  kafka_cluster = module.kafka.kafka_cluster
#  kafka_hosts = module.kafka.kafka_internal_hosts
#  otelcol_endpoint = "${module.balancers.nginx_hosts.app[0]}:${module.telemetry.otelcol_port}"
#}

module "kafka" {
  source = "../../modules/kafka"

  image_resource_group_name = var.image_resource_group_name
  stack = module.network.stack
  cluster = "clusa"
  dns_zone = module.network.dns_zone
  resource_group = module.network.resource_group
  network = module.network.network
  subnet_app = module.network.subnet_app
  bastion_ips = module.network.bastion_ips
  azs = module.network.azs
  public = var.public
}
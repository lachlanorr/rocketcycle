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

module "network" {
  source = "../../modules/network"

  vpc_cidr_block = "10.0.0.0/16"
  stack = "perfa"
  dns_zone = "rkcy.net"
}

module "dev" {
  source = "../../modules/dev"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_edge = module.network.subnet_edge
}

module "kafka" {
  source = "../../modules/kafka"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_app = module.network.subnet_app
  bastion_hosts = module.network.bastion_hosts
}

module "postgresql" {
  source = "../../modules/postgresql"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_storage = module.network.subnet_storage
  bastion_hosts = module.network.bastion_hosts
}
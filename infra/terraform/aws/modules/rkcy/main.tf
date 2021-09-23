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
  type = string
}

variable "vpc_cidr_block" {
  type = string
}

module "network" {
  source = "../../modules/network"

  vpc_cidr_block = var.vpc_cidr_block
  stack = var.stack
  dns_zone = var.dns_zone
}

module "dev" {
  source = "../../modules/dev"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_edge = module.network.subnet_edge
  postgresql_hosts = module.postgresql.postgresql_hosts
  kafka_cluster = module.kafka.kafka_cluster
  kafka_hosts = module.kafka.kafka_hosts
  otelcol_endpoint = module.telemetry.otelcol_endpoint
}

module "kafka" {
  source = "../../modules/kafka"

  stack = module.network.stack
  cluster = "clusa"
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_app = module.network.subnet_app
  bastion_hosts = module.network.bastion_hosts
}

module "telemetry" {
  source = "../../modules/telemetry"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_app = module.network.subnet_app
  bastion_hosts = module.network.bastion_hosts
  elasticsearch_urls = module.elasticsearch.elasticsearch_urls
}

module "metrics" {
  source = "../../modules/metrics"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_app = module.network.subnet_app
  bastion_hosts = module.network.bastion_hosts
  balancer_urls = module.balancers.balancer_urls

  jobs = [
    {
      name = "telemetry_otelcol",
      targets = [for host in module.telemetry.otelcol_hosts: "${host}:9100"]
    },
    {
      name = "telemetry_query",
      targets = [for host in module.telemetry.query_hosts: "${host}:9100"]
    },
    {
      name = "telemetry_collector",
      targets = [for host in module.telemetry.collector_hosts: "${host}:9100"]
    },
    {
      name = "elasticsearch",
      targets = [for host in module.elasticsearch.elasticsearch_hosts: "${host}:9100"]
    },
    {
      name = "kafka",
      targets = [for host in module.kafka.kafka_hosts: "${host}:9100"]
    },
    {
      name = "zookeeper",
      targets = [for host in module.kafka.zookeeper_hosts: "${host}:9100"]
    },
    {
      name = "postgresql",
      targets = [for host in module.postgresql.postgresql_hosts: "${host}:9100"]
    },
    {
      name = "dev",
      targets = [for host in module.dev.dev_hosts: "${host}:9100"]
    },
  ]
}

module "elasticsearch" {
  source = "../../modules/elasticsearch"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_storage = module.network.subnet_storage
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

module "balancers" {
  source = "../../modules/balancers"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_edge = module.network.subnet_edge
  subnet_app = module.network.subnet_app
  bastion_hosts = module.network.bastion_hosts
  jaeger_query_hostports = module.telemetry.jaeger_query_hostports
  prometheus_hostports = module.metrics.prometheus_hostports
  grafana_hostports = module.metrics.grafana_hostports
}

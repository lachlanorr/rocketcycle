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

variable "public" {
  type = bool
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
  kafka_hosts = module.kafka.kafka_internal_hosts
  otelcol_endpoint = "${module.balancers.nginx_hosts.app[0]}:${module.telemetry.otelcol_port}"
}

module "kafka" {
  source = "../../modules/kafka"

  stack = module.network.stack
  cluster = "clusa"
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_app = module.network.subnet_app
  bastion_ips = module.network.bastion_ips
  public = var.public
}

module "telemetry" {
  source = "../../modules/telemetry"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_app = module.network.subnet_app
  bastion_ips = module.network.bastion_ips
  elasticsearch_urls = module.elasticsearch.elasticsearch_urls
  nginx_hosts = module.balancers.nginx_hosts
}

module "metrics" {
  source = "../../modules/metrics"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_app = module.network.subnet_app
  bastion_ips = module.network.bastion_ips
  balancer_internal_urls = module.balancers.balancer_internal_urls
  balancer_external_urls = module.balancers.balancer_external_urls

  jobs = [
    {
      name = "telemetry_otelcol",
      targets = [for host in module.telemetry.otelcol_hosts: "${host}:9100"]
      relabel = [
        {
          source_labels = ["__address__"]
          regex: "([^\\.]+).*"
          target_label = "instance"
          replacement = "$${1}"
        },
      ]
    },
    {
      name = "telemetry_query",
      targets = [for host in module.telemetry.jaeger_query_hosts: "${host}:9100"]
      relabel = [
        {
          source_labels = ["__address__"]
          regex: "([^\\.]+).*"
          target_label = "instance"
          replacement = "$${1}"
        },
      ]
    },
    {
      name = "telemetry_collector",
      targets = [for host in module.telemetry.jaeger_collector_hosts: "${host}:9100"]
      relabel = [
        {
          source_labels = ["__address__"]
          regex: "([^\\.]+).*"
          target_label = "instance"
          replacement = "$${1}"
        },
      ]
    },
    {
      name = "elasticsearch",
      targets = [for host in module.elasticsearch.elasticsearch_hosts: "${host}:9100"]
      relabel = [
        {
          source_labels = ["__address__"]
          regex: "([^\\.]+).*"
          target_label = "instance"
          replacement = "$${1}"
        },
      ]
    },
    {
      name = "kafka",
      targets = [for host in module.kafka.kafka_internal_hosts: "${host}:9100"]
      relabel = [
        {
          source_labels = ["__address__"]
          regex: "([^\\.]+).*"
          target_label = "instance"
          replacement = "$${1}"
        },
        {
          source_labels = ["__address__"]
          regex: "[^\\.]+\\.([^\\.]+).*"
          target_label = "cluster"
          replacement = "$${1}"
        },
      ]
    },
    {
      name = "zookeeper",
      targets = [for host in module.kafka.zookeeper_hosts: "${host}:9100"]
      relabel = [
        {
          source_labels = ["__address__"]
          regex: "([^\\.]+).*"
          target_label = "instance"
          replacement = "$${1}"
        },
        {
          source_labels = ["__address__"]
          regex: "[^\\.]+\\.([^\\.]+).*"
          target_label = "cluster"
          replacement = "$${1}"
        },
      ]
    },
    {
      name = "postgresql",
      targets = [for host in module.postgresql.postgresql_hosts: "${host}:9100"]
      relabel = [
        {
          source_labels = ["__address__"]
          regex: "([^\\.]+).*"
          target_label = "instance"
          replacement = "$${1}"
        },
      ]
    },
    {
      name = "nginx",
      targets = [for host in concat(module.balancers.nginx_hosts.edge, module.balancers.nginx_hosts.edge): "${host}:9100"]
      relabel = [
        {
          source_labels = ["__address__"]
          regex: "([^\\.]+).*"
          target_label = "instance"
          replacement = "$${1}"
        },
        {
          source_labels = ["__address__"]
          regex: "[^\\.]+\\.([^\\.]+).*"
          target_label = "cluster"
          replacement = "$${1}"
        },
      ]
    },
    {
      name = "dev",
      targets = [for host in module.dev.dev_hosts: "${host}:9100"]
      relabel = [
        {
          source_labels = ["__address__"]
          regex: "([^\\.]+).*"
          target_label = "instance"
          replacement = "$${1}"
        },
      ]
    },
  ]
}

module "elasticsearch" {
  source = "../../modules/elasticsearch"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_storage = module.network.subnet_storage
  bastion_ips = module.network.bastion_ips
}

module "postgresql" {
  source = "../../modules/postgresql"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_storage = module.network.subnet_storage
  bastion_ips = module.network.bastion_ips
  public = var.public
}

module "balancers" {
  source = "../../modules/balancers"

  stack = module.network.stack
  dns_zone = module.network.dns_zone
  vpc = module.network.vpc
  subnet_edge = module.network.subnet_edge
  subnet_app = module.network.subnet_app
  bastion_ips = module.network.bastion_ips
  jaeger_query_hosts = module.telemetry.jaeger_query_hosts
  jaeger_query_port = module.telemetry.jaeger_query_port
  jaeger_collector_hosts = module.telemetry.jaeger_collector_hosts
  jaeger_collector_port = module.telemetry.jaeger_collector_port
  otelcol_hosts = module.telemetry.otelcol_hosts
  otelcol_port = module.telemetry.otelcol_port
  prometheus_hosts = module.metrics.prometheus_hosts
  prometheus_port = module.metrics.prometheus_port
  grafana_hosts = module.metrics.grafana_hosts
  grafana_port = module.metrics.grafana_port
  public = var.public
}

module "network" {
  source = "../../modules/network"

  cidr_block = var.cidr_block
  image_resource_group_name = var.image_resource_group_name
  stack = var.stack
  dns_zone = var.dns_zone
}

module "kafka" {
  source = "../../modules/kafka"

  image_resource_group_name = var.image_resource_group_name
  stack = module.network.stack
  cluster = "clusa"
  dns_zone = module.network.dns_zone
  resource_group = module.network.resource_group
  network_cidr = module.network.network.address_space[0]
  subnets = module.network.subnets_app
  bastion_ip = module.network.bastion_ips[0]
  azs = module.network.azs
  public = var.public
}

module "elasticsearch" {
  source = "../../modules/elasticsearch"

  image_resource_group_name = var.image_resource_group_name
  stack = module.network.stack
  dns_zone = module.network.dns_zone
  resource_group = module.network.resource_group
  network_cidr = module.network.network.address_space[0]
  subnets = module.network.subnets_storage
  azs = module.network.azs
  bastion_ip = module.network.bastion_ips[0]
}

module "postgresql" {
  source = "../../modules/postgresql"

  image_resource_group_name = var.image_resource_group_name
  stack = module.network.stack
  dns_zone = module.network.dns_zone
  resource_group = module.network.resource_group
  network_cidr = module.network.network.address_space[0]
  subnets = module.network.subnets_storage
  azs = module.network.azs
  bastion_ip = module.network.bastion_ips[0]
  public = var.public
}

module "telemetry" {
  source = "../../modules/telemetry"

  image_resource_group_name = var.image_resource_group_name
  stack = module.network.stack
  dns_zone = module.network.dns_zone
  resource_group = module.network.resource_group
  network_cidr = module.network.network.address_space[0]
  subnets = module.network.subnets_app
  azs = module.network.azs
  bastion_ip = module.network.bastion_ips[0]
  elasticsearch_urls = module.elasticsearch.elasticsearch_urls
  nginx_telem_host = module.balancers.nginx_hosts.app[0]
}

module "metrics" {
  source = "../../modules/metrics"

  image_resource_group_name = var.image_resource_group_name
  stack = module.network.stack
  dns_zone = module.network.dns_zone
  resource_group = module.network.resource_group
  network_cidr = module.network.network.address_space[0]
  subnets = module.network.subnets_app
  azs = module.network.azs
  bastion_ip = module.network.bastion_ips[0]
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
      targets = [for host in concat(module.balancers.nginx_hosts.app, module.balancers.nginx_hosts.edge): "${host}:9100"]
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

module "balancers" {
  source = "../../modules/balancers"

  image_resource_group_name = var.image_resource_group_name
  stack = module.network.stack
  dns_zone = module.network.dns_zone
  resource_group = module.network.resource_group
  network_cidr = module.network.network.address_space[0]
  subnets_edge = module.network.subnets_edge
  subnets_app = module.network.subnets_app
  azs = module.network.azs
  bastion_ip = module.network.bastion_ips[0]
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

module "dev" {
  source = "../../modules/dev"

  image_resource_group_name = var.image_resource_group_name
  stack = module.network.stack
  dns_zone = module.network.dns_zone
  resource_group = module.network.resource_group
  network_cidr = module.network.network.address_space[0]
  azs = module.network.azs
  subnets = module.network.subnets_edge
  postgresql_hosts = module.postgresql.postgresql_hosts
  kafka_cluster = module.kafka.kafka_cluster
  kafka_hosts = module.kafka.kafka_internal_hosts
  otelcol_endpoint = "${module.balancers.nginx_hosts.app[0]}:${module.telemetry.otelcol_port}"
}

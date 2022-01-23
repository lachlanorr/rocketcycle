data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
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

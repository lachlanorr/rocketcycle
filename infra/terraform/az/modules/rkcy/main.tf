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
  network = module.network.network
  subnet_app = module.network.subnet_app
  bastion_ips = module.network.bastion_ips
  azs = module.network.azs
  public = var.public
}

module "elasticsearch" {
  source = "../../modules/elasticsearch"

  image_resource_group_name = var.image_resource_group_name
  stack = module.network.stack
  dns_zone = module.network.dns_zone
  resource_group = module.network.resource_group
  network = module.network.network
  subnet_storage = module.network.subnet_storage
  azs = module.network.azs
  bastion_ips = module.network.bastion_ips
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


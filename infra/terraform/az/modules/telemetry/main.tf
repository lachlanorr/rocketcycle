data "azurerm_image" "telemetry" {
  name_regex          = "^rkcy-telemetry-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
}

#-------------------------------------------------------------------------------
# jaeger_collector
#-------------------------------------------------------------------------------
module "jaeger_collector_vm" {
  source = "../vm"

  name = "jaeger_collector"
  stack = var.stack
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [for i in range(var.jaeger_collector_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].address_prefixes[0], 14)
    }
  ]

  image_id = data.azurerm_image.telemetry.id
  size = "Standard_D2d_v4"
  public = false

  in_rules = [
    {
      name  = "ssh"
      cidrs = [ var.network_cidr ]
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = [ var.network_cidr ]
      port  = 9100
    },
    {
      name  = "jaeger_collector_grpc"
      cidrs = [ var.network_cidr ]
      port  = var.jaeger_collector_port
    },
    {
      name  = "jaeger_collector_http"
      cidrs = [ var.network_cidr ]
      port  = 14268
    },
    {
      name  = "jaeger_collector_admin"
      cidrs = [ var.network_cidr ]
      port  = 14269
    },
  ]
}

module "jaeger_collector_configure" {
  source = "../../../shared/telemetry/jaeger_collector"
  count = var.jaeger_collector_count

  hostname = module.jaeger_collector_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  jaeger_collector_ip = module.jaeger_collector_vm.vms[count.index].ip
  elasticsearch_urls = var.elasticsearch_urls
}
#-------------------------------------------------------------------------------
# jaeger_collector (END)
#-------------------------------------------------------------------------------

#-------------------------------------------------------------------------------
# jaeger_query
#-------------------------------------------------------------------------------
module "jaeger_query_vm" {
  source = "../vm"

  name = "jaeger_query"
  stack = var.stack
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [for i in range(var.jaeger_query_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].address_prefixes[0], 16)
    }
  ]

  image_id = data.azurerm_image.telemetry.id
  size = "Standard_D2d_v4"
  public = false

  in_rules = [
    {
      name  = "ssh"
      cidrs = [ var.network_cidr ]
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = [ var.network_cidr ]
      port  = 9100
    },
    {
      name  = "jaeger_query_grpc"
      cidrs = [ var.network_cidr ]
      port  = 16685
    },
    {
      name  = "jaeger_query_http"
      cidrs = [ var.network_cidr ]
      port  = var.jaeger_query_port
    },
    {
      name  = "jaeger_query_admin"
      cidrs = [ var.network_cidr ]
      port  = 16687
    },
  ]
}

module "jaeger_query_configure" {
  source = "../../../shared/telemetry/jaeger_query"
  count = var.jaeger_query_count

  hostname = module.jaeger_query_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  jaeger_query_ip = module.jaeger_query_vm.vms[count.index].ip
  elasticsearch_urls = var.elasticsearch_urls
}
#-------------------------------------------------------------------------------
# jaeger_query (END)
#-------------------------------------------------------------------------------

#-------------------------------------------------------------------------------
# otelcol
#-------------------------------------------------------------------------------
module "otelcol_vm" {
  source = "../vm"

  name = "otelcol"
  stack = var.stack
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [for i in range(var.otelcol_count) :
   {
     subnet_id = var.subnets[i % length(var.subnets)].id
     ip = cidrhost(var.subnets[i % length(var.subnets)].address_prefixes[0], 43)
   }
  ]

  image_id = data.azurerm_image.telemetry.id
  size = "Standard_D2d_v4"
  public = false

  in_rules = [
    {
      name  = "ssh"
      cidrs = [ var.network_cidr ]
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = [ var.network_cidr ]
      port  = 9100
    },
    {
      cidrs = [ var.network_cidr ]
      name  = "otelcol_grpc"
      port  = var.otelcol_port
    },
    {
      cidrs = [ var.network_cidr ]
      name  = "otelcol_http"
      port  = 4318
    },
    {
      cidrs = [ var.network_cidr ]
      name  = "otelcol_prometheus_scrape"
      port  = 8888
    },
    {
      cidrs = [ var.network_cidr ]
      name  = "otelcol_prometheus_exporter"
      port  = 9999
    },
    {
      cidrs = [ var.network_cidr ]
      name  = "otelcol_http_legacy"
      port  = 55681
    },
  ]
}

module "otelcol_configure" {
  source = "../../../shared/telemetry/otelcol"
  count = var.otelcol_count

  hostname = module.otelcol_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  otelcol_ip = module.otelcol_vm.vms[count.index].ip
  elasticsearch_urls = var.elasticsearch_urls
  nginx_telem_host = var.nginx_telem_host
  jaeger_collector_port = var.jaeger_collector_port
}
#-------------------------------------------------------------------------------
# otelcol (END)
#-------------------------------------------------------------------------------

data "azurerm_image" "metrics" {
  name_regex          = "^rkcy-metrics-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
}

#-------------------------------------------------------------------------------
# prometheus
#-------------------------------------------------------------------------------
module "prometheus_vm" {
  source = "../vm"

  name = "prometheus"
  stack = var.stack
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [for i in range(var.prometheus_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].address_prefixes[0], 90)
    }
  ]

  image_id = data.azurerm_image.metrics.id
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
      name  = "prometheus"
      cidrs = [ var.network_cidr ]
      port  = var.prometheus_port
    },
  ]
}

module "prometheus_configure" {
  source = "../../../shared/metrics/prometheus"
  count = var.prometheus_count

  hostname = module.prometheus_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path

  jobs = var.jobs
  balancer_url = var.balancer_external_urls.edge
  prometheus_port = var.prometheus_port
  prometheus_ip = module.prometheus_vm.vms[count.index].ip
  prometheus_hosts = sort(module.prometheus_vm.vms[*].hostname)
  grafana_hosts = sort(module.grafana_vm.vms[*].hostname)
}
#-------------------------------------------------------------------------------
# prometheus (END)
#-------------------------------------------------------------------------------

#-------------------------------------------------------------------------------
# grafana
#-------------------------------------------------------------------------------
module "grafana_vm" {
  source = "../vm"

  name = "grafana"
  stack = var.stack
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [for i in range(var.grafana_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].address_prefixes[0], 30)
    }
  ]

  image_id = data.azurerm_image.metrics.id
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
      name  = "grafana"
      cidrs = [ var.network_cidr ]
      port  = var.grafana_port
    },
  ]
}

module "grafana_configure" {
  source = "../../../shared/metrics/grafana"
  count = var.grafana_count

  hostname = module.grafana_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path

  balancer_external_url = var.balancer_external_urls.edge
  balancer_internal_url = var.balancer_internal_urls.app
  grafana_port = var.grafana_port
  grafana_ip = module.grafana_vm.vms[count.index].ip
}
#-------------------------------------------------------------------------------
# grafana (END)
#-------------------------------------------------------------------------------

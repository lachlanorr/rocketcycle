data "azurerm_image" "nginx" {
  name_regex          = "^rkcy-nginx-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  ingress_cidrs = sort(distinct(var.public ? [ var.network_cidr, "${chomp(data.http.myip.body)}/32"] : [ var.network_cidr ]))
  ingress_80443_cidrs = sort(distinct(var.public ? [ var.inbound_cidr, "${chomp(data.http.myip.body)}/32"] : [ var.inbound_cidr ]))
}

module "nginx_vm" {
  source = "../vm"

  name = "nginx"
  stack = var.stack
  cluster = {
    name = var.cluster
    aggregate = true
  }
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [for i in range(var.nginx_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].address_prefixes[0], 80)
    }
  ]

  image_id = data.azurerm_image.nginx.id
  size = "Standard_D2d_v4"
  public = var.public

  in_rules = [
    {
      name  = "ssh"
      cidrs = local.ingress_cidrs
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = local.ingress_cidrs
      port  = 9100
    },
    {
      name  = "nginx_http"
      cidrs = local.ingress_80443_cidrs
      port  = 80
    },
    {
      name  = "nginx_https"
      cidrs = local.ingress_80443_cidrs
      port  = 443
    },
    {
      name  = "nginx_otelcol"
      cidrs = local.ingress_cidrs
      port  = 4317
    },
    {
      name  = "nginx_jaeger_collector"
      cidrs = local.ingress_cidrs
      port  = 14250
    },
  ]
}

module "nginx_configure" {
  source = "../../../shared/nginx"
  count = var.nginx_count

  hostname = module.nginx_vm.vms[0].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  nginx_ip = module.nginx_vm.vms[count.index].ip
  routes = var.routes
}

data "azurerm_image" "postgresql" {
  name_regex          = "^rkcy-postgresql-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  ingress_cidrs = var.public ? [ var.network_cidr, "${chomp(data.http.myip.body)}/32"] : [ var.network_cidr ]
}

module "postgresql_vm" {
  source = "../vm"

  name = "postgresql"
  stack = var.stack
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [for i in range(var.postgresql_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].address_prefixes[0], 54)
    }
  ]

  image_id = data.azurerm_image.postgresql.id
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
      name  = "postgresql"
      cidrs = local.ingress_cidrs
      port  = 5432
    },
  ]
}

module "postgresql_configure" {
  source = "../../../shared/postgresql"
  count = var.postgresql_count

  hostname = module.postgresql_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  postgresql_ip = module.postgresql_vm.vms[count.index].ip
}

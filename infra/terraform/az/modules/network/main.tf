resource "azurerm_resource_group" "rkcy" {
  name     = "rkcy_${var.stack}"
  location = "centralus"
}

resource "azurerm_virtual_network" "rkcy" {
  name                = "rkcy_${var.stack}"
  address_space       = [var.cidr_block]
  location            = azurerm_resource_group.rkcy.location
  resource_group_name = azurerm_resource_group.rkcy.name
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  azs = [1, 2, 3]
}

resource "azurerm_subnet" "rkcy_edge" {
  count                = var.edge_subnet_count
  name                 = "rkcy_${var.stack}_edge_${count.index}"
  resource_group_name  = azurerm_resource_group.rkcy.name
  virtual_network_name = azurerm_virtual_network.rkcy.name
  address_prefixes     = [cidrsubnet(azurerm_virtual_network.rkcy.address_space[0], 8, 0 + count.index)]
}

resource "azurerm_subnet" "rkcy_app" {
  count                = var.app_subnet_count
  name                 = "rkcy_${var.stack}_app_${count.index}"
  resource_group_name  = azurerm_resource_group.rkcy.name
  virtual_network_name = azurerm_virtual_network.rkcy.name
  address_prefixes     = [cidrsubnet(azurerm_virtual_network.rkcy.address_space[0], 8, 100 + count.index)]
}

resource "azurerm_subnet" "rkcy_storage" {
  count                = var.storage_subnet_count
  name                 = "rkcy_${var.stack}_storage_${count.index}"
  resource_group_name  = azurerm_resource_group.rkcy.name
  virtual_network_name = azurerm_virtual_network.rkcy.name
  address_prefixes     = [cidrsubnet(azurerm_virtual_network.rkcy.address_space[0], 8, 200 + count.index)]
}

locals {
  bastion_private_ips = [for i in range(var.bastion_count) : "${cidrhost(azurerm_subnet.rkcy_edge[i].address_prefixes[0], 10)}" ]
}

data "azurerm_image" "bastion" {
  name_regex          = "^rkcy-bastion-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
}

module "bastion_vm" {
  source = "../vm"

  name = "bastion"
  stack = var.stack
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = azurerm_resource_group.rkcy
  network_cidr = azurerm_virtual_network.rkcy.address_space[0]
  dns_zone = var.dns_zone
  azs = local.azs

  vms = [for i in range(var.bastion_count) :
    {
      subnet_id = azurerm_subnet.rkcy_edge[i % length(azurerm_subnet.rkcy_edge)].id
      ip = cidrhost(azurerm_subnet.rkcy_edge[i].address_prefixes[0], 10)
    }
  ]

  image_id = data.azurerm_image.bastion.id
  size = "Standard_D2d_v4"
  public = true

  in_rules = [
    {
      name  = "ssh"
      cidrs = [ "${chomp(data.http.myip.body)}/32", azurerm_virtual_network.rkcy.address_space[0] ]
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = [ azurerm_virtual_network.rkcy.address_space[0] ]
      port  = 9100
    },
  ]
}

module "bastion_configure" {
  source = "../../../shared/network"
  count = var.bastion_count

  hostname = module.bastion_vm.vms[count.index].hostname
  bastion_ip = module.bastion_vm.vms[count.index].public_ip
  ssh_key_path = var.ssh_key_path
}

locals {
  sn_ids   = var.subnet_storage.*.id
  sn_cidrs = flatten(var.subnet_storage.*.address_prefixes)
}

data "azurerm_image" "elasticsearch" {
  name_regex          = "^rkcy-elasticsearch-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
}

locals {
  elasticsearch_racks = [for i in range(var.elasticsearch_count) : "${var.azs[i % var.elasticsearch_count]}"]
  elasticsearch_ips = [for i in range(var.elasticsearch_count) : "${cidrhost(local.sn_cidrs[i], 92)}"]
  elasticsearch_nodes = [for i in range(var.elasticsearch_count) : "master-${i}"]
}

resource "azurerm_network_security_group" "elasticsearch" {
  name                = "rkcy_${var.stack}_elasticsearch"
  location            = var.resource_group.location
  resource_group_name = var.resource_group.name
}

resource "azurerm_network_security_rule" "elasticsearch_ssh" {
  name                        = "AllowSshInbound"
  priority                    = 100
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "22"
  source_address_prefix       = var.network.address_space[0]
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.elasticsearch.name
}

resource "azurerm_network_security_rule" "elasticsearch_node_exporter_in" {
  name                        = "AllowNodeExporterInbound"
  priority                    = 101
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "9100"
  source_address_prefix       = var.network.address_space[0]
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.elasticsearch.name
}

resource "azurerm_network_security_rule" "elasticsearch_rest_in" {
  name                        = "AllowElasticsearchRestInbound"
  priority                    = 102
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "${var.elasticsearch_port}"
  source_address_prefix       = var.network.address_space[0]
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.elasticsearch.name
}

resource "azurerm_network_security_rule" "elasticsearch_nodes_in" {
  name                        = "AllowElasticsearchNodesInbound"
  priority                    = 103
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "9300"
  source_address_prefix       = var.network.address_space[0]
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.elasticsearch.name
}

resource "azurerm_network_security_rule" "elasticsearch_deny_vnet_in" {
  name                       = "DenyVnetInBound"
  priority                   = 200
  direction                  = "Inbound"
  access                     = "Deny"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "*"
  source_address_prefix      = var.network.address_space[0]
  destination_address_prefix = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.elasticsearch.name
}

resource "azurerm_network_interface" "elasticsearch" {
  count               = var.elasticsearch_count
  name                = "rkcy_${var.stack}_elasticsearch_${count.index}"
  location            = var.resource_group.location
  resource_group_name = var.resource_group.name

  ip_configuration {
    name                          = "rkcy_${var.stack}_elasticsearch_${count.index}"
    subnet_id                     = var.subnet_storage[count.index % length(var.subnet_storage)].id
    private_ip_address_allocation = "Static"
    private_ip_address            = local.elasticsearch_ips[count.index]
  }
}

resource "azurerm_network_interface_security_group_association" "elasticsearch" {
  count                     = var.elasticsearch_count
  depends_on                = [azurerm_network_interface.elasticsearch, azurerm_network_security_group.elasticsearch]
  network_interface_id      = azurerm_network_interface.elasticsearch[count.index].id
  network_security_group_id = azurerm_network_security_group.elasticsearch.id
}

resource "azurerm_user_assigned_identity" "elasticsearch" {
  resource_group_name = var.resource_group.name
  location            = var.resource_group.location
  name                = "rkcy_${var.stack}_elasticsearch"
}

resource "azurerm_linux_virtual_machine" "elasticsearch" {
  count                 = var.elasticsearch_count
  depends_on            = [azurerm_network_interface_security_group_association.elasticsearch]
  name                  = "rkcy_${var.stack}_elasticsearch_${count.index}"
  location              = var.resource_group.location
  resource_group_name   = var.resource_group.name
  network_interface_ids = [azurerm_network_interface.elasticsearch[count.index].id]
  size                  = "Standard_D2_v3"
  zone                  = var.azs[count.index % length(var.azs)]

  source_image_id = data.azurerm_image.elasticsearch.id

  identity {
    type = "SystemAssigned, UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.elasticsearch.id]
  }

  computer_name = "elasticsearch-${count.index}"
  os_disk {
    name                 = "rkcy_${var.stack}_elasticsearch_${count.index}"
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  disable_password_authentication = true

  admin_username = "ubuntu"
  admin_ssh_key {
    public_key = file("${var.ssh_key_path}.pub")
    username = "ubuntu"
  }
}

resource "azurerm_dns_a_record" "elasticsearch_private" {
  count               = var.elasticsearch_count
  name                = "elasticsearch-${count.index}.${var.stack}.local"
  zone_name           = var.dns_zone
  resource_group_name = var.image_resource_group_name
  ttl                 = 300
  records             = [local.elasticsearch_ips[count.index]]
}

module "elasticsearch_configure" {
  source = "../../../shared/elasticsearch"
  count = var.elasticsearch_count
  depends_on = [azurerm_linux_virtual_machine.elasticsearch]

  hostname = azurerm_dns_a_record.elasticsearch_private[count.index].fqdn
  bastion_ip = var.bastion_ips[0]
  ssh_key_path = var.ssh_key_path
  stack = var.stack
  elasticsearch_index = count.index
  elasticsearch_ips = local.elasticsearch_ips
  elasticsearch_nodes = local.elasticsearch_nodes
  elasticsearch_rack = local.elasticsearch_racks[count.index]
}

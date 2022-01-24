locals {
  sn_ids   = var.subnet_storage.*.id
  sn_cidrs = flatten(var.subnet_storage.*.address_prefixes)
}

data "azurerm_image" "postgresql" {
  name_regex          = "^rkcy-postgresql-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  ingress_cidrs = var.public ? [ var.network.address_space[0], "${chomp(data.http.myip.body)}/32"] : [ var.network.address_space[0] ]
}

locals {
  postgresql_ips = [for i in range(var.postgresql_count) : "${cidrhost(local.sn_cidrs[i], 100)}"]
}

resource "azurerm_network_security_group" "postgresql" {
  name                = "rkcy_${var.stack}_postgresql"
  location            = var.resource_group.location
  resource_group_name = var.resource_group.name
}

resource "azurerm_network_security_rule" "postgresql_ssh" {
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
  network_security_group_name = azurerm_network_security_group.postgresql.name
}

resource "azurerm_network_security_rule" "postgresql_node_exporter_in" {
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
  network_security_group_name = azurerm_network_security_group.postgresql.name
}

resource "azurerm_network_security_rule" "postgresql_client_in" {
  name                        = "AllowPostgresqlClientInbound"
  priority                    = 102
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "5432"
  source_address_prefix       = var.network.address_space[0]
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.postgresql.name
}

resource "azurerm_network_security_rule" "postgresql_deny_vnet_in" {
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
  network_security_group_name = azurerm_network_security_group.postgresql.name
}

resource "azurerm_network_interface" "postgresql" {
  count               = var.postgresql_count
  name                = "rkcy_${var.stack}_postgresql_${count.index}"
  location            = var.resource_group.location
  resource_group_name = var.resource_group.name

  ip_configuration {
    name                          = "rkcy_${var.stack}_postgresql_${count.index}"
    subnet_id                     = var.subnet_storage[count.index % length(var.subnet_storage)].id
    private_ip_address_allocation = "Static"
    private_ip_address            = local.postgresql_ips[count.index]
    public_ip_address_id          = var.public ? azurerm_public_ip.postgresql[count.index].id : null
  }
}

resource "azurerm_network_interface_security_group_association" "postgresql" {
  count                     = var.postgresql_count
  depends_on                = [azurerm_network_interface.postgresql, azurerm_network_security_group.postgresql]
  network_interface_id      = azurerm_network_interface.postgresql[count.index].id
  network_security_group_id = azurerm_network_security_group.postgresql.id
}

resource "azurerm_user_assigned_identity" "postgresql" {
  resource_group_name = var.resource_group.name
  location            = var.resource_group.location
  name                = "rkcy_${var.stack}_postgresql"
}

resource "azurerm_linux_virtual_machine" "postgresql" {
  count                 = var.postgresql_count
  depends_on            = [azurerm_network_interface_security_group_association.postgresql]
  name                  = "rkcy_${var.stack}_postgresql_${count.index}"
  location              = var.resource_group.location
  resource_group_name   = var.resource_group.name
  network_interface_ids = [azurerm_network_interface.postgresql[count.index].id]
  size                  = "Standard_D2_v3"
  zone                  = var.azs[count.index % length(var.azs)]

  source_image_id = data.azurerm_image.postgresql.id

  identity {
    type = "SystemAssigned, UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.postgresql.id]
  }

  computer_name = "postgresql-${count.index}"
  os_disk {
    name                 = "rkcy_${var.stack}_postgresql_${count.index}"
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

resource "azurerm_public_ip" "postgresql" {
  count               = var.public ? var.postgresql_count : 0
  name                = "rkcy_${var.stack}_postgresql_${count.index}"
  resource_group_name = var.resource_group.name
  location            = var.resource_group.location
  allocation_method   = "Static"
  sku                 = "Standard"
  availability_zone = var.azs[count.index % length(var.azs)]
}

resource "azurerm_dns_a_record" "postgresql_public" {
  count               = var.public ? var.postgresql_count : 0
  name                = "postgresql-${count.index}.${var.stack}"
  zone_name           = var.dns_zone
  resource_group_name = var.image_resource_group_name
  ttl                 = 300
  records             = [azurerm_public_ip.postgresql[count.index].ip_address]
#  target_resource_id  = azurerm_public_ip.postgresql[count.index].id
}

resource "azurerm_dns_a_record" "postgresql_private" {
  count               = var.postgresql_count
  name                = "postgresql-${count.index}.${var.stack}.local"
  zone_name           = var.dns_zone
  resource_group_name = var.image_resource_group_name
  ttl                 = 300
  records             = [local.postgresql_ips[count.index]]
}

module "postgresql_configure" {
  source = "../../../shared/postgresql"
  count = var.postgresql_count
  depends_on = [azurerm_linux_virtual_machine.postgresql]

  hostname = azurerm_dns_a_record.postgresql_private[count.index].fqdn
  bastion_ip = var.bastion_ips[0]
  ssh_key_path = var.ssh_key_path
  postgresql_ip = local.postgresql_ips[count.index]
}

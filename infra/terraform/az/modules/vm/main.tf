locals {
  clusterDot = var.cluster.name != null ? "${var.cluster.name}." : ""
  clusterUnderscore = var.cluster.name != null ? "${var.cluster.name}_" : ""
  fullname = "rkcy_${local.clusterUnderscore}${var.stack}_${var.name}"
  fullnameIdxs = [for i in range(length(var.vms)) : "${local.fullname}_${i}"]
  nameDashes = replace(var.name, "_", "-")
  records = [for i in range(length(var.vms)) : "${local.nameDashes}-${i}.${local.clusterDot}${var.stack}.local"]
  hostnames = [for i in range(length(var.vms)) : "${local.records[i]}.${var.dns_zone}"]
  public_records = [for i in range(length(var.vms)) : "${local.nameDashes}-${i}.${local.clusterDot}${var.stack}"]
  public_hostnames = [for i in range(length(var.vms)) : "${local.public_records[i]}.${var.dns_zone}"]
}

resource "azurerm_network_security_group" "vm" {
  name                = local.fullname
  location            = var.resource_group.location
  resource_group_name = var.resource_group.name
}

resource "azurerm_network_security_rule" "vm_allow" {
  count                       = length(var.in_rules)
  name                        = var.in_rules[count.index].name
  priority                    = 100 + count.index
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "${var.in_rules[count.index].port}"
  source_address_prefixes     = var.in_rules[count.index].cidrs
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.vm.name
}

resource "azurerm_network_security_rule" "vm_deny" {
  name                       = "deny_all_else"
  priority                   = 200
  direction                  = "Inbound"
  access                     = "Deny"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "*"
  source_address_prefix      = var.network_cidr
  destination_address_prefix = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.vm.name
}

resource "azurerm_network_interface" "vm" {
  count               = length(var.vms)
  name                = "${local.fullname}_${count.index}"
  location            = var.resource_group.location
  resource_group_name = var.resource_group.name

  ip_configuration {
    name                          = "${local.fullname}_${count.index}"
    subnet_id                     = var.vms[count.index].subnet_id
    private_ip_address_allocation = "Static"
    private_ip_address            = var.vms[count.index].ip
    public_ip_address_id          = var.public ? azurerm_public_ip.vm[count.index].id : null
  }
}

resource "azurerm_network_interface_security_group_association" "vm" {
  count                     = length(var.vms)
  depends_on                = [azurerm_network_interface.vm, azurerm_network_security_group.vm]
  network_interface_id      = azurerm_network_interface.vm[count.index].id
  network_security_group_id = azurerm_network_security_group.vm.id
}

resource "azurerm_user_assigned_identity" "vm" {
  resource_group_name = var.resource_group.name
  location            = var.resource_group.location
  name                = "${local.fullname}"
}

resource "azurerm_linux_virtual_machine" "vm" {
  count                 = length(var.vms)
  depends_on            = [azurerm_network_interface_security_group_association.vm]
  name                  = "${local.fullname}_${count.index}"
  location              = var.resource_group.location
  resource_group_name   = var.resource_group.name
  network_interface_ids = [azurerm_network_interface.vm[count.index].id]
  size                  = var.size
  zone                  = var.azs[count.index % length(var.azs)]

  source_image_id = var.image_id

  identity {
    type = "SystemAssigned, UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.vm.id]
  }

  computer_name = "${local.nameDashes}-${count.index}"
  os_disk {
    name                 = "${local.fullname}_${count.index}"
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

resource "azurerm_dns_a_record" "private" {
  count               = length(var.vms)
  zone_name           = var.dns_zone
  name                = local.records[count.index]
  resource_group_name = var.dns_resource_group_name
  ttl                 = 300
  records             = [var.vms[count.index].ip]
}

resource "azurerm_dns_a_record" "private_aggregate" {
  count               = var.cluster.aggregate ? 1 : 0
  zone_name           = var.dns_zone
  name                = "${var.cluster.name}.${var.stack}.local"
  resource_group_name = var.dns_resource_group_name
  ttl                 = 300
  records             = var.vms[*].ip
}

resource "azurerm_public_ip" "vm" {
  count               = var.public ? length(var.vms) : 0
  name                = "${local.fullname}_${count.index}"
  resource_group_name = var.resource_group.name
  location            = var.resource_group.location
  allocation_method   = "Static"
  sku                 = "Standard"
  availability_zone   = var.azs[count.index % length(var.azs)]
}

resource "azurerm_dns_a_record" "public" {
  count               = var.public ? length(var.vms) : 0
  zone_name           = var.dns_zone
  name                = local.public_records[count.index]
  resource_group_name = var.dns_resource_group_name
  ttl                 = 300
  records             = [azurerm_public_ip.vm[count.index].ip_address]
}

resource "azurerm_dns_a_record" "public_aggregate" {
  count               = var.public && var.cluster.aggregate ? 1 : 0
  zone_name           = var.dns_zone
  name                = "${var.cluster.name}.${var.stack}"
  resource_group_name = var.dns_resource_group_name
  ttl                 = 300
  records             = azurerm_public_ip.vm[*].ip_address
}

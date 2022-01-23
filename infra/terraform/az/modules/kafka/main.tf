locals {
  sn_ids   = var.subnet_app.*.id
  sn_cidrs = flatten(var.subnet_app.*.address_prefixes)
}

data "azurerm_image" "kafka" {
  name_regex          = "^rkcy-kafka-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  ingress_cidrs = var.public ? [ var.network.address_space[0], "${chomp(data.http.myip.body)}/32"] : [ var.network.address_space[0] ]
}

#-------------------------------------------------------------------------------
# Zookeepers
#-------------------------------------------------------------------------------
locals {
  zookeeper_ips = [for i in range(var.zookeeper_count) : "${cidrhost(local.sn_cidrs[i], 100)}"]
}

resource "azurerm_network_security_group" "zookeeper" {
  name                = "rkcy_${var.cluster}_${var.stack}_zookeeper"
  location            = var.resource_group.location
  resource_group_name = var.resource_group.name

  security_rule = []
}

resource "azurerm_network_security_rule" "zookeeper_ssh" {
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
  network_security_group_name = azurerm_network_security_group.zookeeper.name
}

resource "azurerm_network_security_rule" "zookeeper_node_exporter_in" {
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
  network_security_group_name = azurerm_network_security_group.zookeeper.name
}

resource "azurerm_network_security_rule" "zookeeper_client" {
  name                        = "AllowZookeeperClientInbound"
  priority                    = 102
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "2181"
  source_address_prefix       = var.network.address_space[0]
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.zookeeper.name
}

resource "azurerm_network_security_rule" "zookeeper_peer" {
  name                        = "AllowZookeeperPeerInbound"
  priority                    = 103
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "2888"
  source_address_prefix       = var.network.address_space[0]
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.zookeeper.name
}

resource "azurerm_network_security_rule" "zookeeper_leader" {
  name                        = "AllowZookeeperLeaderInbound"
  priority                    = 104
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "3888"
  source_address_prefix       = var.network.address_space[0]
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.zookeeper.name
}

resource "azurerm_network_security_rule" "zookeeper_deny_vnet_in" {
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
  network_security_group_name = azurerm_network_security_group.zookeeper.name
}

resource "azurerm_network_interface" "zookeeper" {
  count               = var.zookeeper_count
  name                = "rkcy_${var.cluster}_${var.stack}_zookeeper_${count.index}"
  location            = var.resource_group.location
  resource_group_name = var.resource_group.name

  ip_configuration {
    name                          = "rkcy_${var.cluster}_${var.stack}_zookeeper_${count.index}"
    subnet_id                     = var.subnet_app[count.index % length(var.subnet_app)].id
    private_ip_address_allocation = "Static"
    private_ip_address            = local.zookeeper_ips[count.index]
  }
}

resource "azurerm_network_interface_security_group_association" "zookeeper" {
  count                     = var.zookeeper_count
  depends_on                = [azurerm_network_interface.zookeeper, azurerm_network_security_group.zookeeper]
  network_interface_id      = azurerm_network_interface.zookeeper[count.index].id
  network_security_group_id = azurerm_network_security_group.zookeeper.id
}

resource "azurerm_linux_virtual_machine" "zookeeper" {
  count                 = var.zookeeper_count
  depends_on            = [azurerm_network_interface_security_group_association.zookeeper]
  name                  = "rkcy_${var.cluster}_${var.stack}_zookeeper_${count.index}"
  location              = var.resource_group.location
  resource_group_name   = var.resource_group.name
  network_interface_ids = [azurerm_network_interface.zookeeper[count.index].id]
  size                  = "Standard_D2_v3"
  zone                  = var.azs[count.index % length(var.azs)]

  source_image_id = data.azurerm_image.kafka.id

  computer_name = "zookeeper-${count.index}"
  os_disk {
    name                 = "rkcy_${var.cluster}_${var.stack}_zookeeper_${count.index}"
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

resource "azurerm_dns_a_record" "zookeeper_private" {
  count               = var.zookeeper_count
  name                = "zookeeper-${count.index}.${var.cluster}.${var.stack}.local"
  zone_name           = var.dns_zone
  resource_group_name = var.image_resource_group_name
  ttl                 = 300
  records             = [local.zookeeper_ips[count.index]]
}

module "zookeeper_configure" {
  source = "../../../shared/kafka/zookeeper"
  count = var.zookeeper_count
  depends_on = [azurerm_linux_virtual_machine.zookeeper]

  hostname = azurerm_dns_a_record.zookeeper_private[count.index].fqdn
  bastion_ip = var.bastion_ips[0]
  ssh_key_path = var.ssh_key_path
  zookeeper_index = count.index
  zookeeper_ips = local.zookeeper_ips
}
#-------------------------------------------------------------------------------
# Zookeepers (END)
#-------------------------------------------------------------------------------


#-------------------------------------------------------------------------------
# Brokers
#-------------------------------------------------------------------------------
locals {
  kafka_racks = [for i in range(var.kafka_count) : "${var.azs[i % length(var.azs)]}"]
  kafka_internal_ips = [for i in range(var.kafka_count) : "${cidrhost(local.sn_cidrs[i], 101)}"]
  kafka_internal_hosts = [for i in range(var.kafka_count) : "kafka-${i}.${var.cluster}.${var.stack}.local.${var.dns_zone}"]
  kafka_external_ips = azurerm_public_ip.kafka.*.ip_address
  kafka_external_hosts = [for i in range(var.kafka_count) : "kafka-${i}.${var.cluster}.${var.stack}.${var.dns_zone}"]
}

resource "azurerm_network_security_group" "kafka" {
  name                = "rkcy_${var.cluster}_${var.stack}_kafka"
  location            = var.resource_group.location
  resource_group_name = var.resource_group.name

  security_rule = []
}

resource "azurerm_network_security_rule" "kafka_ssh" {
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
  network_security_group_name = azurerm_network_security_group.kafka.name
}

resource "azurerm_network_security_rule" "kafka_node_exporter_in" {
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
  network_security_group_name = azurerm_network_security_group.kafka.name
}

resource "azurerm_network_security_rule" "kafka_internal" {
  name                        = "AllowKafkaInternalInbound"
  priority                    = 102
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "9092"
  source_address_prefix       = var.network.address_space[0]
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.kafka.name
}

resource "azurerm_network_security_rule" "kafka_external" {
  name                        = "AllowKafkaExternalInbound"
  priority                    = 103
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "9093"
  source_address_prefix       = var.network.address_space[0]
  destination_address_prefix  = "*"
  resource_group_name         = var.resource_group.name
  network_security_group_name = azurerm_network_security_group.kafka.name
}

resource "azurerm_network_security_rule" "kafka_deny_vnet_in" {
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
  network_security_group_name = azurerm_network_security_group.kafka.name
}

resource "azurerm_network_interface" "kafka" {
  count               = var.kafka_count
  name                = "rkcy_${var.cluster}_${var.stack}_kafka_${count.index}"
  location            = var.resource_group.location
  resource_group_name = var.resource_group.name

  ip_configuration {
    name                          = "rkcy_${var.cluster}_${var.stack}_kafka_${count.index}"
    subnet_id                     = var.subnet_app[count.index % length(var.subnet_app)].id
    private_ip_address_allocation = "Static"
    private_ip_address            = local.kafka_internal_ips[count.index]
    public_ip_address_id          = var.public ? azurerm_public_ip.kafka[count.index].id : null
  }
}

resource "azurerm_network_interface_security_group_association" "kafka" {
  count                     = var.kafka_count
  depends_on                = [azurerm_network_interface.kafka, azurerm_network_security_group.kafka]
  network_interface_id      = azurerm_network_interface.kafka[count.index].id
  network_security_group_id = azurerm_network_security_group.kafka.id
}

resource "azurerm_linux_virtual_machine" "kafka" {
  count                 = var.kafka_count
  depends_on            = [azurerm_network_interface_security_group_association.kafka]
  name                  = "rkcy_${var.cluster}_${var.stack}_kafka_${count.index}"
  location              = var.resource_group.location
  resource_group_name   = var.resource_group.name
  network_interface_ids = [azurerm_network_interface.kafka[count.index].id]
  size                  = "Standard_D2_v3"
  zone                  = var.azs[count.index % length(var.azs)]

  source_image_id = data.azurerm_image.kafka.id

  computer_name = "kafka-${count.index}"
  os_disk {
    name                 = "rkcy_${var.cluster}_${var.stack}_kafka_${count.index}"
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

resource "azurerm_public_ip" "kafka" {
  count               = var.public ? var.kafka_count : 0
  name                = "rkcy_${var.cluster}_${var.stack}_kafka_${count.index}"
  resource_group_name = var.resource_group.name
  location            = var.resource_group.location
  allocation_method   = "Static"
  sku                 = "Standard"
  availability_zone = var.azs[count.index % length(var.azs)]
}

resource "azurerm_dns_a_record" "kafka_public" {
  count               = var.public ? var.kafka_count : 0
  name                = "kafka-${count.index}.${var.cluster}.${var.stack}"
  zone_name           = var.dns_zone
  resource_group_name = var.image_resource_group_name
  ttl                 = 300
  records             = [azurerm_public_ip.kafka[count.index].ip_address]
#  target_resource_id  = azurerm_public_ip.kafka[count.index].id
}

resource "azurerm_dns_a_record" "kafka_private" {
  count               = var.kafka_count
  name                = "kafka-${count.index}.${var.cluster}.${var.stack}.local"
  zone_name           = var.dns_zone
  resource_group_name = var.image_resource_group_name
  ttl                 = 300
  records             = [local.kafka_internal_ips[count.index]]
}

module "kafka_configure" {
  source = "../../../shared/kafka"
  count = var.kafka_count
  depends_on = [
    azurerm_linux_virtual_machine.kafka,
    module.zookeeper_configure,
  ]

  hostname = azurerm_dns_a_record.kafka_private[count.index].fqdn
  bastion_ip = var.bastion_ips[0]
  ssh_key_path = var.ssh_key_path
  public = var.public

  kafka_index = count.index
  kafka_rack = local.kafka_racks[count.index]
  kafka_internal_ips = local.kafka_internal_ips
  kafka_internal_host = local.kafka_internal_hosts[count.index]
  kafka_external_host = local.kafka_external_hosts[count.index]

  zookeeper_ips = local.zookeeper_ips
}
#-------------------------------------------------------------------------------
# Brokers (END)
#-------------------------------------------------------------------------------

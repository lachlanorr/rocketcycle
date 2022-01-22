terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "=2.91.0"
    }
  }
}

provider "azurerm" {
  features {}
}

variable "image_resource_group" {
  type = string
}

variable "stack" {
  type = string
}

variable "dns_zone" {
  type = string
}

variable "cidr_block" {
  type = string
  default = "10.0.0.0/16"
}

# Put something like this in ~/.ssh/config:
#
# Host bastion-0.rkcy.net
#    User ubuntu
#    IdentityFile ~/.ssh/rkcy_id_rsa
variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "bastion_count" {
  type = number
  default = 1
}

variable "edge_subnet_count" {
  type = number
  default = 3
}

variable "app_subnet_count" {
  type = number
  default = 5
}

variable "storage_subnet_count" {
  type = number
  default = 5
}

variable "location" {
  type = string
  default = "centralus"
}

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

resource "azurerm_network_security_group" "bastion" {
  name                = "rkcy_${var.stack}_bastion"
  location            = azurerm_resource_group.rkcy.location
  resource_group_name = azurerm_resource_group.rkcy.name

  security_rule = []
}

resource "azurerm_network_security_rule" "bastion_ssh" {
  name                        = "AllowSshInbound"
  priority                    = 100
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "22"
  source_address_prefix       = "${chomp(data.http.myip.body)}/32"
  destination_address_prefix  = "*"
  resource_group_name         = azurerm_resource_group.rkcy.name
  network_security_group_name = azurerm_network_security_group.bastion.name
}

resource "azurerm_network_security_rule" "bastion_node_exporter_in" {
  name                       = "AllowNodeExporterInBound"
  priority                   = 101
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "Tcp"
  source_port_range          = "*"
  destination_port_range     = "9100"
  source_address_prefix      = azurerm_virtual_network.rkcy.address_space[0]
  destination_address_prefix = "*"
  resource_group_name         = azurerm_resource_group.rkcy.name
  network_security_group_name = azurerm_network_security_group.bastion.name
}

resource "azurerm_network_security_rule" "bastion_deny_vnet_in" {
  name                       = "DenyVnetInBound"
  priority                   = 200
  direction                  = "Inbound"
  access                     = "Deny"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "*"
  source_address_prefix      = azurerm_virtual_network.rkcy.address_space[0]
  destination_address_prefix = "*"
  resource_group_name         = azurerm_resource_group.rkcy.name
  network_security_group_name = azurerm_network_security_group.bastion.name
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
  count                = var.edge_subnet_count
  name                 = "rkcy_${var.stack}_app_${count.index}"
  resource_group_name  = azurerm_resource_group.rkcy.name
  virtual_network_name = azurerm_virtual_network.rkcy.name
  address_prefixes     = [cidrsubnet(azurerm_virtual_network.rkcy.address_space[0], 8, 100 + count.index)]
}

resource "azurerm_subnet" "rkcy_storage" {
  count                = var.edge_subnet_count
  name                 = "rkcy_${var.stack}_storage_${count.index}"
  resource_group_name  = azurerm_resource_group.rkcy.name
  virtual_network_name = azurerm_virtual_network.rkcy.name
  address_prefixes     = [cidrsubnet(azurerm_virtual_network.rkcy.address_space[0], 8, 200 + count.index)]
}

locals {
  bastion_private_ips = [for i in range(var.bastion_count) : "${cidrhost(azurerm_subnet.rkcy_edge[i].address_prefixes[0], 10)}" ]
}

resource "azurerm_network_interface" "bastion" {
  count               = var.bastion_count
  name                = "rkcy_${var.stack}_bastion_${count.index}"
  location            = azurerm_resource_group.rkcy.location
  resource_group_name = azurerm_resource_group.rkcy.name

  ip_configuration {
    name                          = "rkcy_${var.stack}_bastion_${count.index}"
    subnet_id                     = azurerm_subnet.rkcy_edge[count.index].id
    private_ip_address_allocation = "Static"
    private_ip_address            = local.bastion_private_ips[count.index]
    public_ip_address_id          = azurerm_public_ip.bastion[count.index].id
  }
}

resource "azurerm_network_interface_security_group_association" "bastion" {
  count                     = var.bastion_count
  depends_on                = [azurerm_network_interface.bastion, azurerm_network_security_group.bastion]
  network_interface_id      = azurerm_network_interface.bastion[count.index].id
  network_security_group_id = azurerm_network_security_group.bastion.id
}

data "azurerm_image" "bastion" {
  name_regex          = "^rkcy-bastion-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group
}

resource "azurerm_virtual_machine" "bastion" {
  count                 = var.bastion_count
  depends_on            = [azurerm_network_interface_security_group_association.bastion]
  name                  = "rkcy_${var.stack}_bastion_${count.index}"
  location              = azurerm_resource_group.rkcy.location
  resource_group_name   = azurerm_resource_group.rkcy.name
  network_interface_ids = [azurerm_network_interface.bastion[count.index].id]
  vm_size               = "Standard_D2_v3"
  zones                 = [local.azs[count.index % length(local.azs)]]

  delete_os_disk_on_termination = true

  delete_data_disks_on_termination = true

  storage_image_reference {
    id = data.azurerm_image.bastion.id
  }

  storage_os_disk {
    name              = "rkcy_${var.stack}_bastion_${count.index}"
    caching           = "ReadWrite"
    create_option     = "FromImage"
    managed_disk_type = "Standard_LRS"
  }

  os_profile {
    computer_name = "bastion-${count.index}"
    admin_username = "ubuntu"
  }

  os_profile_linux_config {
    disable_password_authentication = true
    ssh_keys {
      key_data = file("${var.ssh_key_path}.pub")
      path = "/home/ubuntu/.ssh/authorized_keys"
    }
  }
}

resource "azurerm_public_ip" "bastion" {
  count               = var.bastion_count
  name                = "rkcy_${var.stack}_bastion_${count.index}"
  resource_group_name = azurerm_resource_group.rkcy.name
  location            = azurerm_resource_group.rkcy.location
  allocation_method   = "Static"
  sku                 = "Standard"
  availability_zone = local.azs[count.index % length(local.azs)]
}

resource "azurerm_dns_a_record" "bastion_public" {
  count               = var.bastion_count
  name                = "bastion-${count.index}.${var.stack}"
  zone_name           = var.dns_zone
  resource_group_name = var.image_resource_group
  ttl                 = 300
  target_resource_id  = azurerm_public_ip.bastion[count.index].id
}

resource "azurerm_dns_a_record" "bastion_private" {
  count               = var.bastion_count
  name                = "bastion-${count.index}.${var.stack}.local"
  zone_name           = var.dns_zone
  resource_group_name = var.image_resource_group
  ttl                 = 300
  records             = [local.bastion_private_ips[count.index]]
}

resource "null_resource" "bastion_provisioner" {
  count = var.bastion_count
  depends_on = [azurerm_virtual_machine.bastion]

  #---------------------------------------------------------
  # node_exporter
  #---------------------------------------------------------
  provisioner "remote-exec" {
    inline = ["sudo hostnamectl set-hostname bastion-${count.index}.${var.stack}.local.${var.dns_zone}"]
  }
  provisioner "file" {
    content = templatefile("${path.module}/../../../shared/node_exporter_install.sh", {})
    destination = "/home/ubuntu/node_exporter_install.sh"
  }
  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo bash /home/ubuntu/node_exporter_install.sh
rm /home/ubuntu/node_exporter_install.sh
EOF
    ]
  }
  #---------------------------------------------------------
  # node_exporter (END)
  #---------------------------------------------------------

  provisioner "file" {
    source = var.ssh_key_path
    destination = "~/.ssh/id_rsa"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod 600 ~/.ssh/id_rsa"
    ]
  }

  connection {
    type     = "ssh"
    user     = "ubuntu"
    host     = azurerm_public_ip.bastion[count.index].ip_address
    private_key = file(var.ssh_key_path)
  }
}

output "stack" {
  value = var.stack
}

output "resource_group" {
  value = azurerm_resource_group.rkcy
}

output "dns_zone" {
  value = var.dns_zone
}

output "subnet_edge" {
  value = azurerm_subnet.rkcy_edge
}

output "subnet_app" {
  value = azurerm_subnet.rkcy_app
}

output "subnet_storage" {
  value = azurerm_subnet.rkcy_storage
}

output "bastion_ips" {
  value = azurerm_public_ip.bastion.*.ip_address
}

output "availability_zones" {
  value = local.azs
}


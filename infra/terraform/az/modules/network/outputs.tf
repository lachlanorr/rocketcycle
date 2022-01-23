output "stack" {
  value = var.stack
}

output "resource_group" {
  value = azurerm_resource_group.rkcy
}

output "network" {
  value = azurerm_virtual_network.rkcy
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

output "azs" {
  value = local.azs
}

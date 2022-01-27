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

output "subnets_edge" {
  value = azurerm_subnet.rkcy_edge
}

output "subnets_app" {
  value = azurerm_subnet.rkcy_app
}

output "subnets_storage" {
  value = azurerm_subnet.rkcy_storage
}

output "bastion_ips" {
  value = module.bastion_vm.vms[*].public_ip
}

output "azs" {
  value = local.azs
}

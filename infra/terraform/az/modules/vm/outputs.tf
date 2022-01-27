output "vms" {
  value = [for i in range(length(var.vms)) : merge(var.vms[i], {
        ip = azurerm_linux_virtual_machine.vm[i].private_ip_address # overwrite input ip so we don't output until vm is created
        hostname = local.hostnames[i]
        public_hostname = var.public ? local.public_hostnames[i] : null
        public_ip = var.public ? azurerm_public_ip.vm[i].ip_address : null
      }
    )
  ]
}

output "private_aggregate_hostname" {
  value = var.cluster.aggregate ? azurerm_dns_a_record.private_aggregate[0].fqdn : null
}

output "public_aggregate_hostname" {
  value = var.public && var.cluster.aggregate ? azurerm_dns_a_record.public_aggregate[0].fqdn : null
}

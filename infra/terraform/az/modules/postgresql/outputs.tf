output "postgresql_hosts" {
  value = sort(azurerm_dns_a_record.postgresql_private[*].fqdn)
}

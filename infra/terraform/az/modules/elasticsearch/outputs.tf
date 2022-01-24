output "elasticsearch_urls" {
  value = [for host in sort(azurerm_dns_a_record.elasticsearch_private[*].fqdn): "http://${host}:${var.elasticsearch_port}"]
}

output "elasticsearch_hosts" {
  value = sort(azurerm_dns_a_record.elasticsearch_private[*].fqdn)
}

output "elasticsearch_port" {
  value = var.elasticsearch_port
}

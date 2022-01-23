output "zookeeper_hosts" {
  value = sort(azurerm_dns_a_record.zookeeper_private.*.fqdn)
}

output "kafka_cluster" {
  value = "${var.stack}_${var.cluster}"
}

output "kafka_internal_hosts" {
  value = sort(azurerm_dns_a_record.kafka_private.*.fqdn)
}

output "kafka_external_hosts" {
  value = sort(azurerm_dns_a_record.kafka_public.*.fqdn)
}

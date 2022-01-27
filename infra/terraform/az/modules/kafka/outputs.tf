output "zookeeper_hosts" {
  value = sort(module.zookeeper_vm.vms[*].hostname)
}

output "kafka_cluster" {
  value = "${var.cluster}_${var.stack}"
}

output "kafka_internal_hosts" {
  value = sort(module.kafka_vm.vms[*].hostname)
}

output "kafka_external_hosts" {
  value = sort(module.kafka_vm.vms[*].public_hostname)
}

output "elasticsearch_urls" {
  value = [for host in sort(module.elasticsearch_vm.vms[*].hostname): "http://${host}:${var.elasticsearch_port}"]
}

output "elasticsearch_hosts" {
  value = sort(module.elasticsearch_vm.vms[*].hostname)
}

output "elasticsearch_port" {
  value = var.elasticsearch_port
}

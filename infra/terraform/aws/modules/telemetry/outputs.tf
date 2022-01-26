output "jaeger_collector_hosts" {
  value = sort(module.jaeger_collector_vm.vms[*].hostname)
}
output "jaeger_collector_port" {
  value = var.jaeger_collector_port
}

output "jaeger_query_hosts" {
  value = sort(module.jaeger_query_vm.vms[*].hostname)
}
output "jaeger_query_port" {
  value = var.jaeger_query_port
}

output "otelcol_hosts" {
  value = sort(module.otelcol_vm.vms[*].hostname)
}
output "otelcol_port" {
  value = var.otelcol_port
}

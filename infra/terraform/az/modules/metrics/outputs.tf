output "prometheus_hosts" {
  value = sort(module.prometheus_vm.vms[*].hostname)
}

output "prometheus_port" {
  value = var.prometheus_port
}

output "grafana_hosts" {
  value = sort(module.grafana_vm.vms[*].hostname)
}

output "grafana_port" {
  value = var.grafana_port
}

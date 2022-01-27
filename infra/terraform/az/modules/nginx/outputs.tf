output "balancer_internal_url" {
  value = "http://${module.nginx_vm.private_aggregate_hostname}"
}

output "balancer_external_url" {
  value = var.public ? "http://${module.nginx_vm.public_aggregate_hostname}" : ""
}

output "nginx_hosts" {
  value = sort(module.nginx_vm.vms[*].hostname)
}

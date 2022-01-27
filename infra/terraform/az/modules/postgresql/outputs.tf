output "postgresql_hosts" {
  value = sort(module.postgresql_vm.vms[*].hostname)
}

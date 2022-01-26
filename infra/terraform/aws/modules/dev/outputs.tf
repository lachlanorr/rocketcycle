output "dev_hosts" {
  value = sort(module.dev_vm.vms[*].hostname)
}

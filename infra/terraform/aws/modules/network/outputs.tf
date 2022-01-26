output "stack" {
  value = var.stack
}

output "vpc" {
  value = aws_vpc.rkcy
}

output "dns_zone" {
  value = data.aws_route53_zone.zone
}

output "subnets_edge" {
  value = aws_subnet.rkcy_edge
}

output "subnets_app" {
  value = aws_subnet.rkcy_app
}

output "subnets_storage" {
  value = aws_subnet.rkcy_storage
}

output "bastion_ips" {
  value = module.bastion_vm.vms[*].public_ip
}

output "azs" {
  value = local.azs
}

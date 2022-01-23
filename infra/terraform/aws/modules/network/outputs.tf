output "stack" {
  value = var.stack
}

output "vpc" {
  value = aws_vpc.rkcy
}

output "dns_zone" {
  value = data.aws_route53_zone.zone
}

output "subnet_edge" {
  value = aws_subnet.rkcy_edge
}

output "subnet_app" {
  value = aws_subnet.rkcy_app
}

output "subnet_storage" {
  value = aws_subnet.rkcy_storage
}

output "bastion_ips" {
  value = aws_eip.bastion.*.public_ip
}

output "azs" {
  value = local.azs
}

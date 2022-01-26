output "vms" {
  value = [for i in range(length(var.vms)) : merge(var.vms[i], {
        ip = aws_instance.vm[i].private_ip # overwrite input ip so we don't output until vm is created
        hostname = local.hostnames[i]
        public_hostname = var.public ? local.public_hostnames[i] : null
        public_ip = var.public ? aws_eip.vm[i].public_ip : null
      }
    )
  ]
}

output "private_aggregate_hostname" {
  value = var.cluster.aggregate ? aws_route53_record.private_aggregate[0].name : null
}

output "public_aggregate_hostname" {
  value = var.public && var.cluster.aggregate ? aws_route53_record.public_aggregate[0].name : null
}

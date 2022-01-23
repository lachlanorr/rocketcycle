output "postgresql_hosts" {
  value = sort(aws_route53_record.postgresql_private.*.name)
}

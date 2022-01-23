output "dev_hosts" {
  value = sort(aws_route53_record.dev_private.*.name)
}

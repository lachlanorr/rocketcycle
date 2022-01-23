output "elasticsearch_urls" {
  value = [for host in sort(aws_route53_record.elasticsearch_private.*.name): "http://${host}:${var.elasticsearch_port}"]
}

output "elasticsearch_hosts" {
  value = sort(aws_route53_record.elasticsearch_private.*.name)
}

output "elasticsearch_port" {
  value = var.elasticsearch_port
}

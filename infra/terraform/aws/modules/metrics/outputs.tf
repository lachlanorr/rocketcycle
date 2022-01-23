output "prometheus_hosts" {
  value = sort(aws_route53_record.prometheus_private.*.name)
}

output "prometheus_port" {
  value = var.prometheus_port
}

output "grafana_hosts" {
  value = sort(aws_route53_record.grafana_private.*.name)
}

output "grafana_port" {
  value = var.grafana_port
}

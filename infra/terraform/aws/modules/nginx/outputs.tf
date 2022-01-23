output "balancer_internal_url" {
  value = "http://${aws_route53_record.nginx_private_aggregate.name}"
}

output "balancer_external_url" {
  value = var.public ? "http://${aws_route53_record.nginx_public[0].name}" : ""
}

output "nginx_hosts" {
  value = sort(aws_route53_record.nginx_private.*.name)
}

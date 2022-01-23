output "zookeeper_hosts" {
  value = sort(aws_route53_record.zookeeper_private.*.name)
}

output "kafka_cluster" {
  value = "${var.cluster}_${var.stack}"
}

output "kafka_internal_hosts" {
  value = sort(aws_route53_record.kafka_private.*.name)
}

output "kafka_external_hosts" {
  value = sort(aws_route53_record.kafka_public.*.name)
}

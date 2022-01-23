variable "stack" {
  type = string
}

variable "dns_zone" {
  type = any
}

variable "vpc" {
  type = any
}

variable "subnet_edge" {
  type = any
}

variable "subnet_app" {
  type = any
}

variable "bastion_ips" {
  type = list
}

variable "jaeger_query_hosts" {
  type = list
}
variable "jaeger_query_port" {
  type = number
}

variable "jaeger_collector_hosts" {
  type = list
}
variable "jaeger_collector_port" {
  type = number
}

variable "otelcol_hosts" {
  type = list
}
variable "otelcol_port" {
  type = number
}

variable "prometheus_hosts" {
  type = list
}
variable "prometheus_port" {
  type = number
}

variable "grafana_hosts" {
  type = list
}
variable "grafana_port" {
  type = number
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "edge_count" {
  type = number
  default = 1
}

variable "app_count" {
  type = number
  default = 1
}

variable "public" {
  type = bool
}

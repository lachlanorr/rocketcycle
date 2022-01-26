variable "stack" {
  type = string
}

variable "vpc" {
  type = object({
    id = string
    cidr_block = string
  })
}

variable "subnets_edge" {
  type = list(object({
    id = string
    cidr_block = string
  }))
}

variable "subnets_app" {
  type = list(object({
    id = string
    cidr_block = string
  }))
}

variable "dns_zone" {
  type = object({
    name = string
    zone_id = string
  })
}

variable "bastion_ip" {
  type = string
}

variable "jaeger_query_hosts" {
  type = list(string)
}
variable "jaeger_query_port" {
  type = number
}

variable "jaeger_collector_hosts" {
  type = list(string)
}
variable "jaeger_collector_port" {
  type = number
}

variable "otelcol_hosts" {
  type = list(string)
}
variable "otelcol_port" {
  type = number
}

variable "prometheus_hosts" {
  type = list(string)
}
variable "prometheus_port" {
  type = number
}

variable "grafana_hosts" {
  type = list(string)
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

variable "stack" {
  type = string
}

variable "vpc" {
  type = object({
    id = string
    cidr_block = string
  })
}

variable "subnets" {
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

variable "nginx_telem_host" {
  type = string
}

variable "elasticsearch_urls" {
  type = list(string)
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "jaeger_collector_count" {
  type = number
  default = 1
}

variable "jaeger_query_count" {
  type = number
  default = 1
}

variable "jaeger_collector_port" {
  type = number
  default = 14250
}

variable "jaeger_query_port" {
  type = number
  default = 16686
}

variable "otelcol_count" {
  type = number
  default = 1
}

variable "otelcol_port" {
  type = number
  default = 4317
}

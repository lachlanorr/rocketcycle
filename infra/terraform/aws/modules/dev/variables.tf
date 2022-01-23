variable "stack" {
  type = string
}

variable "vpc" {
  type = any
}

variable "subnet_edge" {
  type = any
}

variable "dns_zone" {
  type = any
}

variable "postgresql_hosts" {
  type = any
}

variable "kafka_cluster" {
  type = string
}

variable "kafka_hosts" {
  type = any
}

variable "otelcol_endpoint" {
  type = string
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

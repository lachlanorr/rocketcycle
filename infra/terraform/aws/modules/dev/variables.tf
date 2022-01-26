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

variable "postgresql_hosts" {
  type = list(string)
}

variable "kafka_cluster" {
  type = string
}

variable "kafka_hosts" {
  type = list(string)
}

variable "otelcol_endpoint" {
  type = string
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

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

variable "balancer_external_urls" {
  type = object({
    edge = string
    app = string
  })
}

variable "balancer_internal_urls" {
  type = object({
    edge = string
    app = string
  })
}

variable "jobs" {
  type = list(object({
    name = string
    targets = list(string)
    relabel = list(object({
      source_labels = list(string)
      regex = string
      target_label = string
      replacement = string
    }))
  }))
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "prometheus_count" {
  type = number
  default = 1
}

variable "prometheus_port" {
  type = number
  default = 9090
}

variable "grafana_count" {
  type = number
  default = 1
}

variable "grafana_port" {
  type = number
  default = 3000
}

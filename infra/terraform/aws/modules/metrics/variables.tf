variable "stack" {
  type = string
}

variable "dns_zone" {
  type = any
}

variable "vpc" {
  type = any
}

variable "subnet_app" {
  type = any
}

variable "bastion_ips" {
  type = list
}

variable "balancer_external_urls" {
  type = any
}

variable "balancer_internal_urls" {
  type = any
}

variable "jobs" {
  type = list
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

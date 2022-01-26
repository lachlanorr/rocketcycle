variable "hostname" {
  type = string
}

variable "bastion_ip" {
  type = string
}

variable "ssh_key_path" {
  type = string
}

variable "balancer_external_url" {
  type = string
}

variable "balancer_internal_url" {
  type = string
}

variable "grafana_port" {
  type = number
}

variable "grafana_ip" {
  type = string
}

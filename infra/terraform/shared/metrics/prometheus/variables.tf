variable "hostname" {
  type = string
}

variable "bastion_ip" {
  type = string
}

variable "ssh_key_path" {
  type = string
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

variable "balancer_url" {
  type = string
}

variable "prometheus_port" {
  type = number
}

variable "prometheus_ip" {
  type = string
}

variable "prometheus_hosts" {
  type = list(string)
}

variable "grafana_hosts" {
  type = list(string)
}

variable "hostname" {
  type = string
}

variable "bastion_ip" {
  type = string
}

variable "ssh_key_path" {
  type = string
}

variable "jaeger_collector_ip" {
  type = string
}

variable "elasticsearch_urls" {
  type = list
}
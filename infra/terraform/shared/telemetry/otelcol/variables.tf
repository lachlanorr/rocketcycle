variable "hostname" {
  type = string
}

variable "bastion_ip" {
  type = string
}

variable "ssh_key_path" {
  type = string
}

variable "otelcol_ip" {
  type = string
}

variable "elasticsearch_urls" {
  type = list
}

variable "nginx_telem_host" {
  type = string
}

variable "jaeger_collector_port" {
  type = number
}

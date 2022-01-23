variable "stack" {
  type = string
}

variable "dns_zone" {
  type = any
}

variable "vpc" {
  type = any
}

variable "subnet_storage" {
  type = any
}

variable "bastion_ips" {
  type = list
}

variable "azs" {
  type = list
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "elasticsearch_count" {
  type = number
  default = 3
}

variable "elasticsearch_port" {
  type = number
  default = 9200
}

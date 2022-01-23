variable "stack" {
  type = string
}

variable "vpc" {
  type = any
}

variable "subnet_storage" {
  type = any
}

variable "dns_zone" {
  type = any
}

variable "bastion_ips" {
  type = list
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "postgresql_count" {
  type = number
  default = 1
}

variable "public" {
  type = bool
}

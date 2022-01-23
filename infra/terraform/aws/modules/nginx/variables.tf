variable "stack" {
  type = string
}

variable "cluster" {
  type = string
}

variable "vpc" {
  type = any
}

variable "subnet" {
  type = any
}

variable "dns_zone" {
  type = any
}

variable "bastion_ips" {
  type = list
}

variable "inbound_cidr" {
  type = string
}

variable "routes" {
  type = any
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "nginx_count" {
  type = number
  default = 1
}

variable "public" {
  type = bool
}

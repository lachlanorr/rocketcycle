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

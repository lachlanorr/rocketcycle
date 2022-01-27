variable "image_resource_group_name" {
  type = string
}

variable "stack" {
  type = string
}

variable "dns_zone" {
  type = string
}

variable "resource_group" {
  type = object({
    name = string
    location = string
  })
}

variable "network_cidr" {
  type = string
}

variable "subnets" {
  type = list(object({
    id = string
    address_prefixes = list(string)
  }))
}

variable "bastion_ip" {
  type = string
}

variable "azs" {
  type = list(string)
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

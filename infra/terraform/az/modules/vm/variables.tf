variable "name" {
  type = string
}

variable "cluster" {
  type = map
  default = {
    name = null
    aggregate = false
  }
}

variable "stack" {
  type = string
}

variable "dns_resource_group_name" {
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

variable "dns_zone" {
  type = string
}

variable "vms" {
  type = list(object({
    subnet_id = string
    ip = string
  }))
}

variable "azs" {
  type = list(string)
}

variable "image_id" {
  type = string
}

variable "size" {
  type = string
}

variable "ssh_key_path" {
  type = string
}

variable "public" {
  type = bool
  default = false
}

variable "aggregate_public" {
  type = bool
  default = false
}

variable "aggregate_private" {
  type = bool
  default = false
}

variable "in_rules" {
  type = list(object({
    name = string
    cidrs = list(string)
    port = number
  }))
}

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

variable "vpc_id" {
  type = string
}

variable "dns_zone" {
  type = object({
    name = string
    zone_id = string
  })
}

variable "vms" {
  type = list(object({
    subnet_id = string
    ip = string
  }))
}

variable "image_id" {
  type = string
}

variable "instance_type" {
  type = string
}

variable "key_name" {
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

variable "out_cidrs" {
  type = list(string)
}

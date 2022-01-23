variable "image_resource_group_name" {
  type = string
}

variable "stack" {
  type = string
}

variable "cluster" {
  type = string
}

variable "resource_group" {
  type = any
}

variable "network" {
  type = any
}

variable "subnet_app" {
  type = any
}

variable "dns_zone" {
  type = string
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

variable "zookeeper_count" {
  type = number
  default = 3
}

variable "kafka_count" {
  type = number
  default = 3
}

variable "public" {
  type = bool
}

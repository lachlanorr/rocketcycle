variable "image_resource_group_name" {
  type = string
}

variable "stack" {
  type = string
}

variable "dns_zone" {
  type = string
}

variable "cidr_block" {
  type = string
  default = "10.0.0.0/16"
}

# Put something like this in ~/.ssh/config:
#
# Host bastion-0.rkcy.net
#    User ubuntu
#    IdentityFile ~/.ssh/rkcy_id_rsa
variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "bastion_count" {
  type = number
  default = 1
}

variable "edge_subnet_count" {
  type = number
  default = 3
}

variable "app_subnet_count" {
  type = number
  default = 5
}

variable "storage_subnet_count" {
  type = number
  default = 5
}

variable "location" {
  type = string
  default = "centralus"
}

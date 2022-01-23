variable "hostname" {
  type = string
}

variable "bastion_ip" {
  type = string
}

variable "ssh_key_path" {
  type = string
}

variable "zookeeper_index" {
  type = number
}

variable "zookeeper_ips" {
  type = list
}
